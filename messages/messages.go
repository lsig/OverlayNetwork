package main

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	pb "github.com/lsig/OverlayNetwork/pb"
)

func handleStdInput(wg *sync.WaitGroup, node *types.NodeInfo, registry *types.Registry) {
	defer wg.Done()
	var input string
	listening := true
	for listening {
		fmt.Scanln(&input)

		switch input {
		case "exit":
			fmt.Println("exiting...")

			deregistration := pb.Deregistration{Id: node.Id, Address: node.Address.ToString()}

			chord := pb.MiniChord{Message: &pb.MiniChord_Deregistration{Deregistration: &deregistration}}
			err := utils.SendMessage(registry.Connection, &chord)
			if err != nil {
				fmt.Printf("ERROR: Error when deregistering: %v\n", err.Error())
			} else {
				listening = false
			}
		case "print":
			fmt.Println("printing...")
		default:
			fmt.Println("unknown...")
		}
	}
}

func CreateListenerNode() (*types.NodeInfo, error) {
	port := utils.GenerateRandomPort()
	node := types.NodeInfo{Address: types.Address{Host: net.ParseIP("127.0.0.1"), Port: uint16(port)}}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port)) // remove "localhost" if used externally. This will trigger annoying firewall prompts however
	if err != nil {
		return nil, fmt.Errorf("error listening: %s", err.Error())
	}
	logger.Infof("Listening on port %d", port)

	node.Listener = listener

	return &node, nil
}

func ConnectToRegistry(registry *types.Registry) error {
	tcpServer, err := net.ResolveTCPAddr("tcp", registry.Address.ToString())
	if err != nil {
		return fmt.Errorf("error creating tcp connection to registry: %s", err.Error())
	}

	connection, err := net.DialTCP("tcp", nil, tcpServer)
	if err != nil {
		return fmt.Errorf("error creating tcp connection to registry: %s", err.Error())
	}
	logger.Infof("registry connection on port: %v", connection.LocalAddr().String())

	registry.Connection = connection
	return nil
}

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

func GetNodeRegistry(registry *types.Registry) (*pb.NodeRegistry, error) {
	logger.Info("Waiting for NodeRegistry packet from registry...")
	nodeRegistry, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		return nil, fmt.Errorf("error receiving NodeRegistry packet from registry: %s", err.Error())
	}
	logger.Infof("Received NodeRegistry packet from registry: %v", nodeRegistry)

	nr, ok := nodeRegistry.GetMessage().(*pb.MiniChord_NodeRegistry)
	if !ok {
		return nil, fmt.Errorf("error when parsing nodeRegistry packet")
	}
	return nr.NodeRegistry, nil
}

func SetupNetwork(nodeRegistry *pb.NodeRegistry, node *types.NodeInfo) (*types.Network, error) {
	network := types.Network{}

	for _, peer := range nodeRegistry.Peers {
		peerAddress, err := utils.GetAddressFromString(peer.Address)
		if err != nil {
			return nil, err
		}
		externalNode := types.ExternalNode{Id: peer.Id, Address: *peerAddress}
		network.RoutingTable = append(network.RoutingTable, externalNode)
	}

	for _, id := range nodeRegistry.Ids {
		if id != node.Id {
			network.Nodes = append(network.Nodes, id)
		}
	}

	logger.Debugf("RoutingTable: %v", network.RoutingTable)
	logger.Debugf("Nodes: %v", network.Nodes)

	return &network, nil
}

func HandleListener(wg *sync.WaitGroup, node *types.NodeInfo) {
	defer wg.Done()

	for {
		conn, err := node.Listener.Accept()
		if err != nil {
			logger.Errorf("error handling incoming connection: %s", err.Error())
		}
		logger.Infof("successful incoming connection with: %s", conn.RemoteAddr().Network())
	}
}

func HandleConnector(wg *sync.WaitGroup, network *types.Network) {
	defer wg.Done()

	for _, peer := range network.RoutingTable {
		tcpServer, err := net.ResolveTCPAddr("tcp", peer.Address.ToString())
		if err != nil {
			logger.Errorf("error creating tcp connection to messaging node: %s", err.Error())
		}
		conn, err := net.DialTCP("tcp", nil, tcpServer)
		if err != nil {
			logger.Errorf("error dialing messaging node: %s", err.Error())
		}
		logger.Infof("Connected to node %d at %s", peer.Id, conn.RemoteAddr().String())
	}
}

func main() {
	registry, err := utils.GetRegistryFromProgramArgs(os.Args)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// create Listener Node
	node, err := CreateListenerNode()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer node.Listener.Close()

	// Connect to registry
	err = ConnectToRegistry(registry)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// send Registration
	registrationResponse, err := Register(node, registry)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	node.Id = registrationResponse.Result

	// wait for Node Registry
	nodeRegistry, err := GetNodeRegistry(registry)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.Debugf("Ids: %v", nodeRegistry.Ids)

	// setup network
	network, err := SetupNetwork(nodeRegistry, node)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	logger.Debugf("network: %v\n", network)

	wg := sync.WaitGroup{}

	wg.Add(3)
	go handleStdInput(&wg, node, registry)
	go HandleListener(&wg, node)
	go HandleConnector(&wg, network)
	wg.Wait()
}
