package registry

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"slices"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
	"google.golang.org/protobuf/proto"
)

const I64SIZE = 8

type Node struct {
	Id           int32
	Address      string
	RoutingTable map[int32]string
}

func NewNode(id int32, address string) *Node {
	return &Node{
		Id:           id,
		Address:      address,
		RoutingTable: map[int32]string{},
	}
}

type Registry struct {
	Nodes         map[int32]*Node
	Keys          []int32
	RTableSize    int
	SetupComplete bool
	Addresses     chan string
	Packets       int
}

func NewRegistry() *Registry {
	return &Registry{
		Nodes:         map[int32]*Node{},
		Keys:          []int32{},
		RTableSize:    0,
		SetupComplete: false,
		Addresses:     make(chan string, 128),
		Packets:       0,
	}
}

func (r *Registry) AddNode(address string) {
	if len(r.Keys) >= 128 {
		logger.Warning("Number of Nodes should not exceed 128")
		return
	}

	id := generateId(r.Nodes)
	node := NewNode(int32(id), address)
	r.Nodes[node.Id] = node
	r.Keys = append(r.Keys, node.Id)

	msg := fmt.Sprintf("Node %d added to overlay network", node.Id)
	logger.Info(msg)
}

func (r *Registry) RemoveNode(id int32) {
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

func (r *Registry) ReceiveMessage(conn net.Conn) (*pb.MiniChord, error) {
	bs := make([]byte, I64SIZE)

	if _, err := io.ReadFull(conn, bs); err != nil {
		return nil, err
	}

	numBytes := int(binary.BigEndian.Uint64(bs))

	data := make([]byte, numBytes)

	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	var message pb.MiniChord
	if err := proto.Unmarshal(data, &message); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Received message from %s", conn.RemoteAddr().String())
	logger.Info(msg)

	return &message, nil
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

func generateId(keys map[int32]*Node) int32 {
	for {
		id := int32(rand.IntN(128))
		if _, ok := keys[id]; !ok {
			return id
		}
	}
}

func deleteKey(keys []int32, id int32) []int32 {
	index := -1
	for i, key := range keys {
		if id == key {
			index = i
			break
		}
	}

	if index != -1 {
		keys = append(keys[:index], keys[index+1:]...)
	}
	return keys
}
