package registry

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
)

const I64SIZE = 8

func (r *Registry) Start() {
	logger.Info("Registry listener started")

	r.MessageProcessing()

	for {
		conn, err := r.Listener.Accept()
		if err != nil {
			msg := fmt.Sprintf("Error accepting connection %s", err)
			logger.Error(msg)
			continue
		}
		go r.HandleConnection(conn)
	}
}

func (r *Registry) CommandLineInterface() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		switch {
		case command == "list":
			r.HandleList()
		case command == "route":
			continue
		case strings.HasPrefix(command, "setup "):
			param := strings.TrimPrefix(command, "setup ")
			n, err := strconv.Atoi(param)
			if err != nil {
				logger.Error("Invalid number of nodes:" + param)
				continue
			}
			r.HandleSetup(n)
		case strings.HasPrefix(command, "start "):
			param := strings.TrimPrefix(command, "start ")
			n, err := strconv.Atoi(param)
			if err != nil {
				logger.Error("Invalid number of Packets:" + param)
				continue
			}
			r.HandleStart(n)
		}
	}

}

func (r *Registry) MessageProcessing() {
	go func() {
		for packet := range r.Packets {
			switch msg := packet.Content.Message.(type) {
			case *pb.MiniChord_Registration:
				r.HandleRegistration(packet.Conn, msg)
			case *pb.MiniChord_Deregistration:
				continue
			case *pb.MiniChord_NodeRegistry:
				r.HandleNodeRegistry()
			case *pb.MiniChord_NodeRegistryResponse:
				continue
			case *pb.MiniChord_InitiateTask:
				r.HandleInitiateTask(packet.Content)
			case *pb.MiniChord_TaskFinished:
				continue
			case *pb.MiniChord_ReportTrafficSummary:
				continue
			default:
				errMsg := fmt.Sprintf("Unknown message type received: %s", msg)
				logger.Error(errMsg)
			}
		}
	}()
}

func (r *Registry) ReceiveMessage(conn net.Conn) error {
	bs := make([]byte, I64SIZE)

	if _, err := io.ReadFull(conn, bs); err != nil {
		return err
	}

	numBytes := int(binary.BigEndian.Uint64(bs))

	data := make([]byte, numBytes)

	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}

	packet := Packet{
		Conn:    conn,
		Content: &pb.MiniChord{},
	}

	if err := proto.Unmarshal(data, packet.Content); err != nil {
		return err
	}

	msg := fmt.Sprintf("Received message from %s", conn.RemoteAddr().String())
	logger.Info(msg)

	r.Packets <- &packet

	return nil
}

func (r *Registry) SendMessage(conn net.Conn, message *pb.MiniChord) error {
	data, err := proto.Marshal(message)

	if err != nil {
		logger.Error("Failed to marshal message")
		return fmt.Errorf("Failed to marshal message %w", err)
	}

	msg := fmt.Sprintf("Sending message to %s", conn.RemoteAddr().String())
	logger.Info(msg)

	bs := make([]byte, I64SIZE)
	binary.BigEndian.PutUint64(bs, uint64(len(data)))

	if _, err := conn.Write(bs); err != nil {
		logger.Error("Error sending length message")
		return fmt.Errorf("Error sending message of length %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		logger.Error("Error sending message data")
		return fmt.Errorf("Error sending message data %w", err)
	}

	return nil
}

func (r *Registry) HandleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		err := r.ReceiveMessage(conn)
		if err != nil {
			if err != io.EOF {
				msg := fmt.Sprintf("Error receiving message: %v", err)
				logger.Error(msg)
			}
			break
		}
	}
}
