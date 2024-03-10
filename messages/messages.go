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
func main() {
	registry, err := utils.GetRegistryFromProgramArgs(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	port := utils.GenerateRandomPort()
	node := types.NodeInfo{Address: net.ParseIP("127.0.0.1"), Port: uint16(port)}
	network := types.Network{}
	logger.Infof("network: %v\n", network)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port)) // remove "localhost" if used externally. This will trigger annoying firewall prompts however
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	defer listener.Close()
	logger.Infof("Listening on port %d", port)

	logger.Infof("registry: %v:%v", registry.Address.String(), strconv.Itoa(int(registry.Port)))
	tcpServer, err := net.ResolveTCPAddr("tcp", registry.Address.String()+":"+strconv.Itoa(int(registry.Port)))
	if err != nil {
		fmt.Println("Error creating tcp connection to registry: \n", err.Error())
		os.Exit(1)
	}
	connection, err := net.DialTCP("tcp", nil, tcpServer)
	if err != nil {
		fmt.Println("Error creating tcp connection to registry: \n", err.Error())
		os.Exit(1)
	}
	logger.Info("connected to registry")
	registry.Connection = connection

	logger.Infof("registry connection on port: %v", connection.LocalAddr().String())
	message := pb.Registration{Address: node.Address.String() + ":" + strconv.Itoa(int(node.Port))}

	chord := pb.MiniChord{Message: &pb.MiniChord_Registration{Registration: &message}}

	utils.SendMessage(registry.Connection, &chord)
	response, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		fmt.Printf("error receiving Registration Response: %s\n", err.Error())
		os.Exit(1)
	}

	logger.Infof("Received minichord response: %v", response)

	logger.Info("Waiting for NodeRegistry packet from registry...")

	nodeRegistry, err := utils.ReceiveMessage(registry.Connection)
	if err != nil {
		logger.Error(fmt.Sprintf("error receiving NodeRegistry packet from registry: %s", err.Error()))
		os.Exit(1)
	}

	logger.Infof("Received NodeRegistry response from registry: %v", nodeRegistry)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go handleStdInput(&wg, &node, registry)
	wg.Wait()
}
