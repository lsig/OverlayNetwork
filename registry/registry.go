package registry

import (
	"fmt"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/node"
)

func NewNode(id uint32, address string) *node.Node {
	return &node.Node{
		Id:           id,
		Address:      address,
		RoutingTable: []int{},
	}
}

type Registry struct {
	Nodes   map[uint32]*node.Node
	NoNodes int
}

func NewRegistry() *Registry {
	return &Registry{
		Nodes:   map[uint32]*node.Node{},
		NoNodes: 0,
	}
}

func (r *Registry) AddNode(address string) {
	if r.NoNodes >= 128 {
		logger.Warning("Number of Nodes should not exceed 128")
		return
	}

	node := NewNode(uint32(r.NoNodes), address)
	r.Nodes[node.Id] = node
	r.NoNodes++

	msg := fmt.Sprintf("Node %d added to overlay network", node.Id)
	logger.Info(msg)
}

func (r *Registry) RemoveNode(id uint32) {
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
