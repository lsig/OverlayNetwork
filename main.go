package main

import (
	"fmt"
	"time"

	"github.com/lsig/OverlayNetwork/registry"
)

func performTasksWithNodes(r *registry.Registry) {
	fmt.Println("Performing tasks with nodes in the network...")
	for id, node := range r.Nodes {
		fmt.Printf("Performing task with node ID: %d, Address: %s\n", id, node.Address)
		time.Sleep(1 * time.Second)
	}
}

func main() {
	r := registry.NewRegistry()

	nodeAddresses := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, address := range nodeAddresses {
		r.AddNode(address)
	}

	// Perform some tasks with the nodes
	performTasksWithNodes(r)

	// Remove nodes from the network
	for id := range r.Nodes {
		r.RemoveNode(id)
	}
	r.RemoveNode(100)
}
