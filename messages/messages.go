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
	"strings"

	minichord "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
)

type Registry struct {
	Address net.IP
	Port    uint16
}

func stdInputListener(messageChan chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		messageChan <- scanner.Text()
	}
}

func getRegistryFromProgramArgs(args []string) (*Registry, error) {
	usageError := fmt.Errorf("usage: go run messages/messages.go <registry-host>:<registry-port>")
	if len(args) != 2 {
		return nil, usageError
	}

	addressInfo := strings.Split(args[1], ":")
	if len(addressInfo) != 2 {
		return nil, usageError
	}

	if addressInfo[0] == "localhost" {
		addressInfo[0] = "0.0.0.0"
	}

	address := net.ParseIP(addressInfo[0])
	port, err := strconv.Atoi(addressInfo[1])

	if address == nil || err != nil || port <= 0 || port >= 65536 {
		return nil, usageError
	}

	registry := Registry{Address: address, Port: uint16(port)}

	return &registry, nil
}

func main() {
	registry, err := getRegistryFromProgramArgs(os.Args)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%v\n", registry)

	fmt.Printf("%v arguments\n", len(os.Args))
	listener, err := net.Listen("tcp", "localhost:8080") // remove "localhost" if used externally. This will trigger annoying firewall prompts however
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}

	// Create a channel to receive std input
	messageChan := make(chan string)
	go stdInputListener(messageChan)

	defer listener.Close()
	fmt.Println("TCP server listening on port 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			return
		}
		fmt.Printf("Accepted connection from client: %v\n", conn.RemoteAddr().String())

		// Handle connection in a goroutine, allowing the server to continue listening.
		go func(c net.Conn) {
			defer c.Close()
			// Read the request
			go func() {
				scanner := bufio.NewScanner(conn)
				for scanner.Scan() {
					fmt.Printf("%s: %s\n", c.RemoteAddr().String(), scanner.Text())
				}
			}()

			for {
				message := <-messageChan
				// Wait for a message from the channel and send it to the client
				msg := strings.TrimSpace(message)
				if msg == "exit" {
					fmt.Println("Closing connection with", c.RemoteAddr().String())
					break
				}
				io.WriteString(c, message+"\n")
			}

		}(conn)
	}
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
