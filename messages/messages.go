package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"

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
	fmt.Printf("Messages here\n")
	listener, err := net.Listen("tcp", ":8080")
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
			// Read the request
			buffer := make([]byte, 1024)
			c.Read(buffer)

			fmt.Printf("client %s: %s", c.RemoteAddr().String(), string(buffer))

			// Wait for a message from the channel and send it to the client
			message := <-messageChan
			io.WriteString(c, message+"\n")

			// Close the connection
			c.Close()
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
