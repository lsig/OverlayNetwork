package registry

import (
	"fmt"
	"math/rand/v2"

	"github.com/lsig/OverlayNetwork/logger"
)

type Node struct {
	Id           int32
	Address      string
	RoutingTable []int
}

func NewNode(id int32, address string) *Node {
	return &Node{
		Id:           id,
		Address:      address,
		RoutingTable: []int{},
	}
}

type Registry struct {
	Nodes    map[int32]*Node
	Keys     []int32
	NoNodes  int
	Capacity int
}

func NewRegistry(capacity int) *Registry {
	return &Registry{
		Nodes:    map[int32]*Node{},
		Keys:     []int32{},
		NoNodes:  0,
		Capacity: capacity,
	}
}

func (r *Registry) AddNode(address string) {
	if r.NoNodes >= 128 {
		logger.Warning("Number of Nodes should not exceed 128")
		return
	}

	id := generateId(r.Nodes)
	node := NewNode(int32(id), address)
	r.Nodes[node.Id] = node
	r.NoNodes++

	msg := fmt.Sprintf("Node %d added to overlay network", node.Id)
	logger.Info(msg)
}

func (r *Registry) RemoveNode(id int32) {
	_, ok := r.Nodes[id]
	if ok {
		delete(r.Nodes, id)

		msg := fmt.Sprintf("Node %d removed from overlay network", id)
		logger.Info(msg)
	} else {
		msg := fmt.Sprintf("Node %d not found", id)
		logger.Error(msg)
	}
}

func generateId(keys map[int32]*Node) int32 {
	for {
		id := int32(rand.IntN(128))
		if _, ok := keys[id]; !ok {
			return id
		}
	}
}

func (r *Registry) SendRegistry() {

}
