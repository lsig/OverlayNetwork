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
	Conn         net.Conn
}

func NewNode(id int32, address string, connection net.Conn) *Node {
	return &Node{
		Id:           id,
		Address:      address,
		RoutingTable: map[int32]string{},
		Conn:         connection,
	}
}

type Packet struct {
	Conn    net.Conn
	Content *pb.MiniChord
}

type Registry struct {
	Nodes         map[int32]*Node
	IdSpace       []int32
	Keys          []int32
	RTableSize    int
	SetupSent     bool
	SetupComplete bool
	StartComplete bool
	NoPackets     int
	NoSetupNodes  int
	NoFinished    int
	Summaries     []Summary
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

	idSpace := []int32{}

	for i := range 128 {
		idSpace = append(idSpace, int32(i))
	}

	return &Registry{
		Nodes:         map[int32]*Node{},
		IdSpace:       idSpace,
		Keys:          []int32{},
		RTableSize:    0,
		SetupSent:     false,
		SetupComplete: false,
		StartComplete: false,
		NoPackets:     0,
		NoSetupNodes:  0,
		Listener:      listener,
		Packets:       make(chan *Packet, 128),
	}, nil
}

type Summary struct {
	Id            int32
	Sent          uint32
	Received      uint32
	Relayed       uint32
	TotalSent     int64
	TotalReceived int64
}
