package registry

import (
	"fmt"
	"math"
	"net"
	"slices"

	"math/rand/v2"

	"github.com/lsig/OverlayNetwork/logger"
)

func (r *Registry) AddNode(address string, connection net.Conn) int32 {
	r.Locker.Lock()
	defer r.Locker.Unlock()
	if len(r.Keys) >= 128 {
		logger.Warning("Number of Nodes should not exceed 128")
		return -1
	}

	id := r.generateId()
	node := NewNode(int32(id), address, connection)
	r.Nodes[node.Id] = node
	r.Keys = append(r.Keys, node.Id)

	msg := fmt.Sprintf("Node %d added to overlay network", node.Id)
	logger.Info(msg)
	return id
}

func (r *Registry) RemoveNode(id int32) int32 {
	r.Locker.Lock()
	defer r.Locker.Unlock()
	_, ok := r.Nodes[id]
	if ok {
		delete(r.Nodes, id)
		r.Keys = deleteKey(r.Keys, id)

		msg := fmt.Sprintf("Node %d removed from overlay network", id)
		logger.Info(msg)
		return id
	} else {
		msg := fmt.Sprintf("Node %d not found", id)
		logger.Error(msg)
		return -1
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
	r.RTableSize = size
}

func (r *Registry) AddressExists(address string) bool {
	for _, node := range r.Nodes {
		if address == node.Address {
			return true
		}
	}
	return false
}

func (r *Registry) generateId() int32 {
	index := rand.IntN(len(r.IdSpace))

	id := r.IdSpace[index]

	r.IdSpace = append(r.IdSpace[:index], r.IdSpace[index+1:]...)
	return id
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

func verifyAddress(clientAddr string, connAddr string) bool {
	clientIp, _, err := net.SplitHostPort(clientAddr)

	if err != nil {
		return false
	}

	connIp, _, err := net.SplitHostPort(connAddr)

	if err != nil {
		return false
	}

	if clientIp == connIp {
		return true
	}

	return false
}
