package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	pb "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
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
			err := SendMiniChordMessage(registry.Connection, &chord)
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
	fmt.Printf("network: %v\n", network)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port)) // remove "localhost" if used externally. This will trigger annoying firewall prompts however
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Printf("Listening on port %d\n", port)

	fmt.Printf("registry: %v:%v\n", registry.Address.String(), strconv.Itoa(int(registry.Port)))
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
	fmt.Println("connected to registry")
	registry.Connection = connection

	message := pb.Registration{Address: node.Address.String()}

	chord := pb.MiniChord{Message: &pb.MiniChord_Registration{Registration: &message}}

	SendMiniChordMessage(registry.Connection, &chord)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go handleStdInput(&wg, &node, registry)
	wg.Wait()
}

const I64SIZE int = 8

func SendMiniChordMessage(conn net.Conn, message *pb.MiniChord) (err error) {
	data, err := proto.Marshal(message)
	log.Printf("SendMiniChordMessage(): sending %s (%v), %d to %s\n", message, data, len(data), conn.RemoteAddr().String())
	if err != nil {
		log.Panicln("Failed to marshal message.", err)
	}

	// First send the number of bytes in the marshaled message
	bs := make([]byte, I64SIZE)
	binary.BigEndian.PutUint64(bs, uint64(len(data)))
	length, err := conn.Write(bs)
	if err != nil {
		log.Printf("SendMiniChordMessage() error: %s\n", err)
	}
	if length != I64SIZE {
		log.Panicln("Short write?")
	}

	// Send the marshales message
	length, err = conn.Write(data)
	if err != nil {
		log.Printf("SendMiniChordMessage() error: %s\n", err)
	}
	if length != len(data) {
		log.Panicln("Short write?")
	}
	return
}

func ReceiveMiniChordMessage(conn net.Conn) (message *pb.MiniChord, err error) { // First, get the number of bytes to received
	bs := make([]byte, I64SIZE)
	length, err := conn.Read(bs)
	if err != nil {
		if err != io.EOF {
			log.Printf("ReceivedMiniChordMessage() read error: %s\n", err)
		}
		return
	}
	if length != I64SIZE {
		log.Printf("ReceivedMiniChordMessage() length error: %d\n", length)
		return
	}
	numBytes := uint64(binary.BigEndian.Uint64(bs))
	// Get the marshaled message from the connection
	data := make([]byte, numBytes)
	length, err = conn.Read(data)
	if err != nil {
		if err != io.EOF {
			log.Printf("ReceivedMiniChordMessage() read error: %s\n", err)
		}
		return
	}
	if uint64(length) != numBytes {
		log.Printf("ReceivedMiniChordMessage() length error: %d\n", length)
		return
	}
	// Unmarshal the message
	message = &pb.MiniChord{}
	err = proto.Unmarshal(data[:length], message)
	if err != nil {
		log.Printf("ReceivedMiniChordMessage() unmarshal error: %s\n",
			err)
		return
	}
	log.Printf("ReceiveMiniChordMessage(): received %s (%v), %d from %s\n",
		message, data[:length], length, conn.RemoteAddr().String())
	return
}
