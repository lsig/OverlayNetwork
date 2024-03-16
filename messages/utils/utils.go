package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/messages/types"
	pb "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
)

func GetAddressFromString(addrString string) (*types.Address, error) {
	addressInfo := strings.Split(addrString, ":")
	if len(addressInfo) != 2 {
		return nil, fmt.Errorf("semicolon missing in address string: %s", addrString)
	}

	if addressInfo[0] == "localhost" {
		addressInfo[0] = "127.0.0.1"
	}

	address := net.ParseIP(addressInfo[0])
	port, err := strconv.Atoi(addressInfo[1])

	if address == nil || err != nil || port <= 1024 || port >= 65536 {
		return nil, fmt.Errorf("invalid address or port")
	}

	return &types.Address{Host: address, Port: uint16(port)}, nil
}

func GetRegistryFromProgramArgs(args []string) (*types.Registry, error) {
	usageError := fmt.Errorf("usage: go run messages/messages.go <registry-host>:<registry-port>")
	if len(args) != 2 {
		return nil, usageError
	}

	address, err := GetAddressFromString(args[1])
	if err != nil {
		return nil, usageError
	}

	registry := types.Registry{Address: *address}

	return &registry, nil
}

func GenerateRandomPort() int {
	randomPort := -1

	for randomPort < 0 {
		randomPort = rand.Intn(int(math.Pow(2, 16))-1024) + 1024 // first ca 1024 ports are restricted for OS
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", randomPort))
		if err == nil {
			fmt.Printf("server live on: %v\n", conn.RemoteAddr().String())
			randomPort = -1
			conn.Close()
		}
	}

	return randomPort
}

const I64SIZE int = 8

func ReceiveMessage(conn net.Conn) (*pb.MiniChord, error) {
	// get length of message
	bs := make([]byte, I64SIZE)
	if _, err := io.ReadFull(conn, bs); err != nil {
		return nil, err
	}
	numBytes := int(binary.BigEndian.Uint64(bs))

	// get the amount of data specified by message length above
	data := make([]byte, numBytes)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	// unmarshal the bytes into a minichord message
	message := &pb.MiniChord{}
	if err := proto.Unmarshal(data, message); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Received %v message from %s", GetMiniChordType(message), conn.RemoteAddr().String())
	logger.Info(msg)

	return message, nil
}

func SendMessage(conn net.Conn, message *pb.MiniChord) error {
	data, err := proto.Marshal(message)

	if err != nil {
		logger.Error("Failed to marshal message")
		return fmt.Errorf("failed to marshal message %w", err)
	}

	// logger.Infof("Sending %s message to %s", GetMiniChordType(message), conn.RemoteAddr().String())

	bs := make([]byte, I64SIZE)
	binary.BigEndian.PutUint64(bs, uint64(len(data)))

	if _, err := conn.Write(bs); err != nil {
		logger.Error("Error sending length message")
		return fmt.Errorf("error sending message of length %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		logger.Error("Error sending message data")
		return fmt.Errorf("error sending message data %w", err)
	}

	return nil
}

func GetMiniChordType(msg *pb.MiniChord) string {
	switch msg.Message.(type) {
	case *pb.MiniChord_Registration:
		return "Registration"
	case *pb.MiniChord_RegistrationResponse:
		return "RegistrationResponse"
	case *pb.MiniChord_Deregistration:
		return "Deregistration"
	case *pb.MiniChord_NodeRegistry:
		return "NodeRegistry"
	case *pb.MiniChord_NodeRegistryResponse:
		return "NodeRegistryResponse"
	case *pb.MiniChord_InitiateTask:
		return "InitiateTasks"
	case *pb.MiniChord_NodeData:
		return "NodeData"
	case *pb.MiniChord_TaskFinished:
		return "TaskFinished"
	case *pb.MiniChord_ReportTrafficSummary:
		return "ReportTrafficSummary"
	default:
		logger.Warning("unknown minichord message encountered...")
		return "Unknown"
	}
}
