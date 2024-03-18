package types

import (
	"net"
	"strconv"

	pb "github.com/lsig/OverlayNetwork/pb"
)

type Address struct {
	Host net.IP
	Port uint16
}

func (a Address) ToString() string {
	return a.Host.String() + ":" + strconv.Itoa(int(a.Port))
}

type Registry struct {
	Address    Address
	Connection net.Conn
}

type NodeInfo struct {
	Id        int32
	Address   Address
	Listener  net.Listener
	Listening bool
	IsSetup   bool
	Stats     pb.TrafficSummary
}

type ExternalNode struct {
	Id         int32
	Address    Address
	Connection net.Conn
}

type Network struct {
	Nodes        []int32
	RoutingTable []*ExternalNode
	SendChannel  chan *pb.NodeData
}
