package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
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

			deregistration := pb.Deregistration{Id: int32(node.ID), Address: node.Address.String()}

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
	node := types.NodeInfo{Address: net.ParseIP("127.0.0.1"), Port: uint16(port)}

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port)) // remove "localhost" if used externally. This will trigger annoying firewall prompts however
	if err != nil {
		return nil, fmt.Errorf("error listening:", err.Error())
	}
	logger.Infof("Listening on port %d", port)

	node.Listener = listener

	return &node, nil
}

func ConnectToRegistry(registry *types.Registry) error {
	tcpServer, err := net.ResolveTCPAddr("tcp", registry.Address.String()+":"+strconv.Itoa(int(registry.Port)))
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

func Register(node *types.NodeInfo, registry *types.Registry) (*pb.MiniChord, error) {
	message := pb.Registration{Address: node.Address.String() + ":" + strconv.Itoa(int(node.Port))}
	chord := pb.MiniChord{Message: &pb.MiniChord_Registration{Registration: &message}}

	utils.SendMessage(registry.Connection, &chord)
	response, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		return nil, fmt.Errorf("error receiving Registration Response: %s", err.Error())
	}
	logger.Infof("Received minichord response: %v", response)

	return response, nil
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
	_, err = Register(node, registry)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// wait for Node Registry
	logger.Info("Waiting for NodeRegistry packet from registry...")
	nodeRegistry, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		logger.Errorf("error receiving NodeRegistry packet from registry: %s", err.Error())
		os.Exit(1)
	}
	logger.Infof("Received NodeRegistry response from registry: %v", nodeRegistry)

	// Creating network
	network := types.Network{}
	logger.Infof("network: %v\n", network)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go handleStdInput(&wg, node, registry)
	wg.Wait()
}
