package main

import (
	"fmt"
	"io"
	"math/rand/v2"
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

	// choose a random buffer size
	// can't see how this would matter a whole lot as there isn't a "perfect" buffer size here
	node.SendChannel = make(chan pb.NodeData, 8)

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
		if err == io.EOF {
			return nil, fmt.Errorf("registry has crashed, shutting down")
		}
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
		network.RoutingTable = append(network.RoutingTable, &externalNode)
	}

	for _, id := range nodeRegistry.Ids {
		if id != node.Id {
			network.Nodes = append(network.Nodes, id)
		}
	}

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
		logger.Infof("successful incoming connection with: %s", conn.RemoteAddr().String())
	}
}

func HandleConnector(wg *sync.WaitGroup, network *types.Network) {
	defer wg.Done()

	// for _, peer := range network.RoutingTable {
	// 	tcpServer, err := net.ResolveTCPAddr("tcp", peer.Address.ToString())
	// 	if err != nil {
	// 		logger.Errorf("error creating tcp connection to messaging node: %s", err.Error())
	// 	}
	// 	conn, err := net.DialTCP("tcp", nil, tcpServer)
	// 	if err != nil {
	// 		logger.Errorf("error dialing messaging node: %s", err.Error())
	// 	} else {
	// 		peer.Connection = conn
	// 		logger.Infof("Connected to node %d at %s", peer.Id, conn.RemoteAddr().String())
	// 	}
	// }
}

func ConnectToNeighbours(network *types.Network) {
	// create a waitgroup, so that the function doesn't exit unless all neighbours have been connected to.
	wg := sync.WaitGroup{}

	// I remember something about dynamically incrementing the waitGroup counter is bad practice..?
	// Therefore, we add the length of the routing table instead of incrementing for each iteration in the for loop below.
	wg.Add(len(network.RoutingTable))

	for _, peer := range network.RoutingTable {
		go func(p *types.ExternalNode, wg *sync.WaitGroup) {
			// dial peer until connection is made
			tcpServer, err := net.ResolveTCPAddr("tcp", p.Address.ToString())
			if err != nil {
				logger.Errorf("error creating tcp connection to messaging node: %s\naborting...", err.Error())
				return
			}
			tries := 10
			for p.Connection == nil && tries >= 0 {
				if tries <= 0 {
					logger.Errorf("Could not connect to neighbour %s", p.Address.ToString())
					break
				}
				conn, err := net.DialTCP("tcp", nil, tcpServer)
				if err != nil {
					// logger.Errorf("error dialing messaging node: %s", err.Error())
				} else {
					p.Connection = conn
					logger.Infof("Connected to node %d at %s", p.Id, conn.RemoteAddr().String())
				}
				tries--
			}
			wg.Done()
		}(peer, &wg)
	}
	wg.Wait()
}

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

func getRandomNode(nodes []int32) int32 {
	index := rand.IntN(len(nodes))
	return nodes[index]
}

func CreatePackets(node *types.NodeInfo, network *types.Network, packets uint32) {
	for range packets {
		logger.Debug("adding packet to channel...")
		packet := pb.NodeData{Destination: getRandomNode(network.Nodes), Source: node.Id, Payload: 1, Hops: 0, Trace: []int32{}}
		node.SendChannel <- packet
	}
	logger.Debugf("%d packets added to channel", packets)
}

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
	if err = ConnectToRegistry(registry); err != nil {
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
	wg.Add(4)

	go HandleListener(&wg, node)
	ConnectToNeighbours(network)

	// Send NodeRegistry Response
	if err = SendNodeRegistryResponse(node, network, registry); err != nil {
		logger.Errorf("error sending NodeRegistryResponse to registry: %s", err.Error())
		os.Exit(1)
	}

	packets, err := GetInitiateTasks(registry)
	if err != nil {
		logger.Errorf("error receiving Initiate Tasks: %s", err.Error())
		os.Exit(1)
	}

	// create and add packets to sendChannel
	go CreatePackets(node, network, packets)

	go HandleRegistry(&wg, registry)
	go HandleConnector(&wg, network)
	go handleStdInput(&wg, node, registry)
	wg.Wait()
}
