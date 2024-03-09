package types

import "net"

type Registry struct {
	Address net.IP
	Port    uint16
}

type NodeInfo struct {
	ID      int
	Address net.IP
	Port    uint16
}

type Network struct {
	Nodes        []int32
	RoutingTable map[int32]string
}
