package helpers

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	pb "github.com/lsig/OverlayNetwork/pb"
)

// connects to the registry using a provided address,
// and stores the connection in the registry struct
func ConnectToRegistry(registry *types.Registry) error {
	tcpServer, err := net.ResolveTCPAddr("tcp", registry.Address.ToString())
	if err != nil {
		return fmt.Errorf("error creating tcp connection to registry: %s", err.Error())
	}

	connection, err := net.DialTCP("tcp", nil, tcpServer)
	if err != nil {
		return fmt.Errorf("error creating tcp connection to registry: %s", err.Error())
	}
	// logger.Info("connected to registry")

	registry.Connection = connection
	return nil
}

// Registers node to registry and gets a node Id
func Register(node *types.NodeInfo, registry *types.Registry) (*pb.RegistrationResponse, error) {
	message := pb.Registration{Address: node.Address.ToString()}
	chord := pb.MiniChord{Message: &pb.MiniChord_Registration{Registration: &message}}

	utils.SendMessage(registry.Connection, &chord)
	response, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		return nil, fmt.Errorf("error receiving Registration Response: %s", err.Error())
	}

	nr, ok := response.GetMessage().(*pb.MiniChord_RegistrationResponse)
	if !ok {
		return nil, fmt.Errorf("error when parsing registrationResponse packet")
	}

	logger.Infof("my Id is: %d", nr.RegistrationResponse.Result)

	return nr.RegistrationResponse, nil
}

// Gets the NodeRegistry request from the registry and returns it.
func GetNodeRegistry(registry *types.Registry) (*pb.NodeRegistry, error) {
	logger.Info("Waiting for NodeRegistry packet from registry...")
	nodeRegistry, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("registry has crashed, shutting down")
		}
		return nil, fmt.Errorf("error receiving NodeRegistry packet from registry: %s", err.Error())
	}
	// logger.Infof("Received NodeRegistry packet from registry: %v", nodeRegistry)

	nr, ok := nodeRegistry.GetMessage().(*pb.MiniChord_NodeRegistry)
	if !ok {
		return nil, fmt.Errorf("error when parsing nodeRegistry packet")
	}
	return nr.NodeRegistry, nil
}

// Checks whether connecting to nodes in routing table succeeded
// then sends outcome in NodeRegistryResponse packet to registry
func SendNodeRegistryResponse(node *types.NodeInfo, network *types.Network, registry *types.Registry) error {
	success := true
	for _, peer := range network.RoutingTable {
		if peer.Connection == nil {
			logger.Errorf("connection to peer %d seems to be nil", peer.Id)
			success = false
		}
	}

	if success {
		response := pb.NodeRegistryResponse{Result: uint32(node.Id), Info: fmt.Sprintf("I, node %v, address %s, hereby confirm that I've successfully connected to all my neigbours...", node.Id, node.Address.ToString())}
		chord := pb.MiniChord{Message: &pb.MiniChord_NodeRegistryResponse{NodeRegistryResponse: &response}}

		return utils.SendMessage(registry.Connection, &chord)
	} else {
		// message nodes can't send -1 below, even though the assignment description specifies that it must do that on failure.
		// as nodes can only have valid ids between 0 - 127, a failure Id can be 128.
		response := pb.NodeRegistryResponse{Result: 128, Info: fmt.Sprintf("I, node %v, address %s, hereby deny that I've successfully connected to all my neigbours...", node.Id, node.Address.ToString())}
		chord := pb.MiniChord{Message: &pb.MiniChord_NodeRegistryResponse{NodeRegistryResponse: &response}}

		return utils.SendMessage(registry.Connection, &chord)
	}
}

// Handles receiving unexpected registry requests.
// Still not sure whether this is needed, as I think all registry-node communication is linear.
func HandleRegistry(wg *sync.WaitGroup, registry *types.Registry) {
	for {
		pbMessage, err := utils.ReceiveMessage(registry.Connection)
		if err != nil {
			if err == io.EOF {
				logger.Error("registry has crashed, shutting down...")
				registry.Connection.Close()
				os.Exit(1)
			}
			logger.Errorf("error when receiving registry message: %s", err.Error())
		} else {
			logger.Debugf("received pbMessage from registry: %v", pbMessage)
		}
	}
}

// Get the InitiateTask MiniChord Message from the registry
func GetInitiateTasks(registry *types.Registry) (uint32, error) {
	chord, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		return 0, err
	}

	nr, ok := chord.GetMessage().(*pb.MiniChord_InitiateTask)
	if !ok {
		return 0, fmt.Errorf("error when parsing registrationResponse packet")
	}

	logger.Infof("Initiate packets: %d", nr.InitiateTask.Packets)

	return nr.InitiateTask.Packets, nil
}

// Waits for all messages to have been sent
// and then sends TaskFinished message to registry
func SendTaskFinishedAndTrafficSummary(packets uint32, node *types.NodeInfo, registry *types.Registry) {
	for packets > node.Stats.Sent {
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("All packets sent, sending TaskFinished")

	taskFinished := &pb.TaskFinished{Id: node.Id, Address: node.Address.ToString()}
	chord := &pb.MiniChord{Message: &pb.MiniChord_TaskFinished{TaskFinished: taskFinished}}

	err := utils.SendMessage(registry.Connection, chord)
	if err != nil {
		logger.Errorf("error sending TaskFinished to registry: %s", err.Error())
		os.Exit(1)
	}

	// Receive RequestTrafficSummary
	response, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		logger.Errorf("error when receiving RequestTrafficSummary: %s", err.Error())
		os.Exit(1)
	}

	switch response.Message.(type) {
	case *pb.MiniChord_RequestTrafficSummary:
		// Send TrafficSummary
		err := SendTrafficSummary(registry, node)
		if err != nil {
			logger.Errorf("error sending TrafficSummary: %s", err.Error())
			os.Exit(1)
		}
	default:
		logger.Errorf("response not of type RequestTrafficSummary: %v", response.Message)
	}
}

func SendTrafficSummary(registry *types.Registry, node *types.NodeInfo) error {

	trafficSummary := &pb.TrafficSummary{Id: node.Id, Sent: node.Stats.Sent, Relayed: node.Stats.Relayed, Received: node.Stats.Received, TotalSent: node.Stats.TotalSent, TotalReceived: node.Stats.TotalReceived}

	chord := &pb.MiniChord{Message: &pb.MiniChord_ReportTrafficSummary{ReportTrafficSummary: trafficSummary}}

	logger.Infof("Sending TrafficSummary: %v", trafficSummary)

	return utils.SendMessage(registry.Connection, chord)
}
