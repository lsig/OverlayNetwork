package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	minichord "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
)

func stdInputListener(messageChan chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		messageChan <- scanner.Text()
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

	message := minichord.Registration{Address: node.Address.String()}

	chord := minichord.MiniChord{Message: &minichord.MiniChord_Registration{Registration: &message}}

	SendMiniChordMessage(registry.Connection, &chord)
}

const I64SIZE int = 8

func SendMiniChordMessage(conn net.Conn, message *minichord.MiniChord) (err error) {
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

func ReceiveMiniChordMessage(conn net.Conn) (message *minichord.MiniChord, err error) { // First, get the number of bytes to received
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
	message = &minichord.MiniChord{}
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
