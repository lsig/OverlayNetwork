package registry

import (
	"net"
	"sync"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
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

type Packet struct {
	Conn    net.Conn
	Content *pb.MiniChord
}

type Registry struct {
	Nodes         map[int32]*Node
	Keys          []int32
	RTableSize    int
	SetupComplete bool
	Addresses     chan string
	NoPackets     int
	Listener      net.Listener
	Packets       chan *Packet
	Locker        sync.Mutex
}

func NewRegistry(port string) (*Registry, error) {
	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		logger.Error("Failed to initilize listener")
		return nil, err
	}

	return &Registry{
		Nodes:         map[int32]*Node{},
		Keys:          []int32{},
		RTableSize:    0,
		SetupComplete: false,
		Addresses:     make(chan string, 128),
		NoPackets:     0,
		Listener:      listener,
		Packets:       make(chan *Packet, 128),
	}, nil
}
