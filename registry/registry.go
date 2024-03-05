package registry

import (
	"fmt"

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
	Nodes   map[int32]*Node
	NoNodes int
}

func NewRegistry() *Registry {
	return &Registry{
		Nodes:   map[int32]*Node{},
		NoNodes: 0,
	}
}

func (r *Registry) AddNode(address string) {
	if r.NoNodes >= 128 {
		logger.Warning("Number of Nodes should not exceed 128")
		return
	}

	node := NewNode(int32(r.NoNodes), address)
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

func (r *Registry) SendRegistry() {

}
