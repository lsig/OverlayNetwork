package types

import "net"

type Registry struct {
	Address    net.IP
	Port       uint16
	Connection net.Conn
}

type NodeInfo struct {
	ID       int
	Address  net.IP
	Port     uint16
	Listener net.Listener
}

type Network struct {
	Nodes        []int32
	RoutingTable map[int32]string
}
