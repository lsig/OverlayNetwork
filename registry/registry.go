package registry

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"slices"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
)

const I64SIZE = 8

func (r *Registry) AddNode(address string) int32 {
	r.Locker.Lock()
	defer r.Locker.Unlock()
	if len(r.Keys) >= 128 {
		logger.Warning("Number of Nodes should not exceed 128")
		return -1
	}

	id := generateId(r.Nodes)
	node := NewNode(int32(id), address)
	r.Nodes[node.Id] = node
	r.Keys = append(r.Keys, node.Id)

	msg := fmt.Sprintf("Node %d added to overlay network", node.Id)
	logger.Info(msg)
	return id
}

func (r *Registry) RemoveNode(id int32) {
	r.Locker.Lock()
	defer r.Locker.Unlock()
	_, ok := r.Nodes[id]
	if ok {
		delete(r.Nodes, id)
		r.Keys = deleteKey(r.Keys, id)

		msg := fmt.Sprintf("Node %d removed from overlay network", id)
		logger.Info(msg)
	} else {
		msg := fmt.Sprintf("Node %d not found", id)
		logger.Error(msg)
	}
}

func (r *Registry) GenerateRoutingTables(size int) {
	r.Locker.Lock()
	defer r.Locker.Unlock()

	slices.Sort(r.Keys)
	noKeys := len(r.Keys)

	for index, key := range r.Keys {
		for i := range size {
			neighbour := int(math.Pow(2, float64(i)))
			neighbourIndex := (index + neighbour) % noKeys
			neighbourKey := r.Keys[neighbourIndex]
			neighbourNode := r.Nodes[neighbourKey]

			node := r.Nodes[key]
			node.RoutingTable[neighbourKey] = neighbourNode.Address
		}
	}

}

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

func (r *Registry) MessageProcessing() {
	go func() {
		for packet := range r.Packets {
			switch msgType := packet.Content.Message.(type) {
			case *pb.MiniChord_Registration:
				continue
			case *pb.MiniChord_Deregistration:
				continue
			case *pb.MiniChord_NodeRegistryResponse:
				continue
			case *pb.MiniChord_TaskFinished:
				continue
			case *pb.MiniChord_ReportTrafficSummary:
				continue
			default:
				errMsg := fmt.Sprintf("Unknown message type received: %s", msgType)
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
