package helpers

import (
	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	pb "github.com/lsig/OverlayNetwork/pb"
)

// creates fake packets and sends onto network channel
func CreatePackets(node *types.NodeInfo, network *types.Network, packets uint32) {
	for range packets {
		// logger.Debug("adding packet to channel...")
		packet := pb.NodeData{Destination: utils.GetRandomNode(network.Nodes), Source: node.Id, Payload: 1, Hops: 0, Trace: []int32{}}
		network.SendChannel <- &packet
	}
	// logger.Debugf("%d packets added to channel", packets)
}
