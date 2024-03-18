package helpers

import (
	"fmt"
	"net"
	"sort"
	"sync"

	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	pb "github.com/lsig/OverlayNetwork/pb"
)

// Creates Listener Node object, containing:
// Id
// Address
// Listener
// Stats
func CreateListenerNode() (*types.NodeInfo, error) {
	port := utils.GenerateRandomPort()
	// hardcoding the IP address only makes sense for this testing environment.
	// With nodes covering multiple addresses, the external IP address should be used.
	node := types.NodeInfo{Address: types.Address{Host: net.ParseIP("127.0.0.1"), Port: uint16(port)}}

	// remove "localhost" if used externally.
	// We explicitly prefix this to avoid firewall prompts on startup
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, fmt.Errorf("error listening: %s", err.Error())
	}
	// logger.Infof("Listening on port %d", port)

	node.Listener = listener
	return &node, nil
}

// Sets up network object containing:
func SetupNetwork(nodeRegistry *pb.NodeRegistry, node *types.NodeInfo) (*types.Network, error) {
	network := types.Network{}

	for _, peer := range nodeRegistry.Peers {
		peerAddress, err := utils.GetAddressFromString(peer.Address)
		if err != nil {
			return nil, err
		}
		externalNode := types.ExternalNode{Id: peer.Id, Address: *peerAddress}
		network.RoutingTable = append(network.RoutingTable, &externalNode)
	}

	for _, id := range nodeRegistry.Ids {
		if id != node.Id {
			network.Nodes = append(network.Nodes, id)
		}
	}

	// IMPORTANT: sort the routing table by ExternalNode.Id
	// No two nodes have the same Id, so no need to use sort.SliceStable
	sort.Slice(network.RoutingTable, func(i, j int) bool {
		return network.RoutingTable[i].Id < network.RoutingTable[j].Id
	})

	// choose a random buffer size
	// can't see how finding the exact size matters a whole lot as there isn't a "perfect" buffer size here
	network.SendChannel = make(chan *pb.NodeData, 8)
	return &network, nil
}

// creates fake packets and sends onto network channel
func CreatePackets(node *types.NodeInfo, network *types.Network, packets uint32) {
	for range packets {
		// logger.Debug("adding packet to channel...")
		packet := pb.NodeData{Destination: utils.GetRandomNode(network.Nodes), Source: node.Id, Payload: utils.GeneratePayload(), Hops: 0, Trace: []int32{}}
		network.SendChannel <- &packet
	}
	// logger.Debugf("%d packets added to channel", packets)
}

// Continuously scans the stdin for user commands
// and performs actions based on the recieved command
func HandleStdInput(wg *sync.WaitGroup, node *types.NodeInfo, registry *types.Registry) {
	defer wg.Done()
	var input string
	listening := true
	for listening {
		fmt.Scanln(&input)

		switch input {
		case "exit":
			fmt.Println("exiting...")

			deregistration := pb.Deregistration{Id: node.Id, Address: node.Address.ToString()}

			chord := pb.MiniChord{Message: &pb.MiniChord_Deregistration{Deregistration: &deregistration}}
			err := utils.SendMessage(registry.Connection, &chord)
			if err != nil {
				fmt.Printf("ERROR: Error when deregistering: %v\n", err.Error())
			} else {
				listening = false
			}
		case "print":
			fmt.Printf("    Sent %d\n", node.Stats.Sent)
			fmt.Printf("Received %d\n", node.Stats.Received)
			fmt.Printf(" Relayed %d\n", node.Stats.Relayed)
			fmt.Printf("Total     Sent %d\n", node.Stats.TotalSent)
			fmt.Printf("Total Received %d\n", node.Stats.TotalReceived)
		default:
			fmt.Println("unknown...")
		}
	}
}
