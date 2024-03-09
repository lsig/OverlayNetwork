package registry

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/lsig/OverlayNetwork/logger"
)

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

func (r *Registry) SendRegistry() {

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
