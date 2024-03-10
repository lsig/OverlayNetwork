package registry

import (
	"fmt"
	"math"
	"net"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
)

func (r *Registry) HandleRegistration(conn net.Conn, msg *pb.MiniChord_Registration) {

	var info string
	success := true

	registrationAddr := msg.Registration.GetAddress()

	if !verifyAddress(registrationAddr, conn.RemoteAddr().String()) {
		success = false
		info = "Registration request unsuccessful: Address mismatch."
	}

	if r.AddressExists(msg.Registration.GetAddress()) {
		success = false
		info = "Registration request unsuccessful: Address already exists."
	}

	id := r.AddNode(registrationAddr, conn)

	if success {
		info = fmt.Sprintf("Registration request successful. The number of messaging nodes currently constituting the overlay is (%d).", len(r.Keys))
		logger.Info(info)
	} else {
		logger.Error(info)
	}

	res := &pb.RegistrationResponse{
		Result: id,
		Info:   info,
	}

	chordMessage := &pb.MiniChord{
		Message: &pb.MiniChord_RegistrationResponse{
			RegistrationResponse: res,
		},
	}

	if err := r.SendMessage(conn, chordMessage); err != nil {
		errMsg := fmt.Sprintf("Failed to send registration response: %v", err)
		logger.Error(errMsg)

		if id != -1 {
			// Remove node if sending response fails
			r.RemoveNode(id)
		}
	}
}

func (r *Registry) HandleNodeRegistry() {
	for _, node := range r.Nodes {
		peers := []*pb.Deregistration{}
		for key, val := range node.RoutingTable {
			info := &pb.Deregistration{
				Id:      key,
				Address: val,
			}
			peers = append(peers, info)
		}
		nodeRegistry := &pb.NodeRegistry{
			NR:    uint32(len(node.RoutingTable)),
			NoIds: uint32(len(r.Keys)),
			Peers: peers,
			Ids:   r.Keys,
		}
		chordMessage := &pb.MiniChord{
			Message: &pb.MiniChord_NodeRegistry{
				NodeRegistry: nodeRegistry,
			},
		}

		if err := r.SendMessage(node.Conn, chordMessage); err != nil {
			errMsg := fmt.Sprintf("Failed to send NodeRegistry request: %v", err)
			logger.Error(errMsg)
		}
		logger.Info(fmt.Sprintf("Succesfully sent NodeRegistry to node %d", node.Id))
	}
	r.SetupComplete = true
}

func (r *Registry) HandleSetup(routingTableSize int) {
	maxSize := math.Floor(math.Log2(float64(len(r.Nodes))))

	if routingTableSize > int(maxSize) {
		routingTableSize = int(maxSize)
	}

	r.GenerateRoutingTables(routingTableSize)

	nodeRegistry := &pb.NodeRegistry{
		NR:    0,
		Peers: []*pb.Deregistration{},
		NoIds: 0,
		Ids:   []int32{},
	}

	miniChordMsg := &pb.MiniChord{
		Message: &pb.MiniChord_NodeRegistry{
			NodeRegistry: nodeRegistry,
		},
	}

	packet := &Packet{
		Conn:    nil,
		Content: miniChordMsg,
	}

	r.Packets <- packet
}
