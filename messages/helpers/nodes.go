package helpers

import (
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/messages/types"
	"github.com/lsig/OverlayNetwork/messages/utils"
	pb "github.com/lsig/OverlayNetwork/pb"
)

func ConnectToNeighbours(network *types.Network) {
	// create a waitgroup, so that the function doesn't exit unless all neighbours have been connected to.
	wg := sync.WaitGroup{}

	// I remember something about dynamically incrementing the waitGroup counter is bad practice..?
	// Therefore, we add the length of the routing table instead of incrementing for each iteration in the for loop below.
	wg.Add(len(network.RoutingTable))

	for _, peer := range network.RoutingTable {
		go func(p *types.ExternalNode, wg *sync.WaitGroup) {
			// dial peer until connection is made
			tcpServer, err := net.ResolveTCPAddr("tcp", p.Address.ToString())
			if err != nil {
				logger.Errorf("error creating tcp connection to messaging node: %s\naborting...", err.Error())
				return
			}
			tries := 10
			for p.Connection == nil && tries >= 0 {
				if tries <= 0 {
					logger.Errorf("Could not connect to neighbour %s", p.Address.ToString())
					break
				}
				conn, err := net.DialTCP("tcp", nil, tcpServer)
				if err != nil {
					// logger.Errorf("error dialing messaging node: %s", err.Error())
				} else {
					p.Connection = conn
					// logger.Infof("Connected to node %d", p.Id)
				}
				tries--
			}
			wg.Done()
		}(peer, &wg)
	}
	wg.Wait()
}

// Handles each receiving connection from other message nodes
// runs in a separate goroutine
func HandleNodeConnection(conn net.Conn, node *types.NodeInfo, network *types.Network) {
	for {
		chord, err := utils.ReceiveMessage(conn)
		if err != nil {
			if err != io.EOF {
				logger.Errorf("error receiving message from node: %s", err.Error())
			} else {
				logger.Infof("Sender node has disconnected")
				conn.Close()
			}
			break
		}

		nr, ok := chord.GetMessage().(*pb.MiniChord_NodeData)
		if !ok {
			logger.Error("error when parsing registrationResponse packet to NodeData")
			break
		}

		nodeData := nr.NodeData

		if malformed, _ := utils.NodeDataPacketIsMalformed(nodeData, node); malformed {
			logger.Warningf("received malformed packet, dropping: %v", nodeData)
			break
		}

		if nodeData.Destination == node.Id {
			node.RecvLock.Lock()
			// this packet is for me!
			node.Stats.Received++
			node.Stats.TotalReceived += int64(nodeData.Payload)
			// logger.Debugf("received NodeData message: %v", nodeData)
			node.RecvLock.Unlock()
		} else {
			node.Stats.Relayed++
			// TODO check if my id appears in the trace.
			nodeData.Trace = append(nodeData.Trace, node.Id)
			// logger.Debugf("relaying NodeData message: %v", nodeData)
			// add to channel in a separate goroutine,
			// as we don't want the existing goroutine to be blocked from receiving new messages
			// if the channel is full
			go func(nw *types.Network, nd *pb.NodeData) {
				nw.SendChannel <- nd
			}(network, nodeData)
		}
	}
}

// Accepts incoming connections from other message nodes
// and creates a goroutine for handling that specific connection
func HandleListener(wg *sync.WaitGroup, node *types.NodeInfo, network *types.Network) {
	defer wg.Done()

	for node.Listening {
		conn, err := node.Listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				logger.Infof("Listener closed. Exiting accept loop.")
				break
			}
			logger.Errorf("error handling incoming connection: %s", err.Error())
		}
		// logger.Infof("successful incoming connection with: %s", conn.RemoteAddr().String())
		go HandleNodeConnection(conn, node, network)
	}
	logger.Info("Node is no longer listening")
}

// Receives packets from the packet channel
// and finds the optimal neighbour to send to
func HandleConnector(wg *sync.WaitGroup, node *types.NodeInfo, network *types.Network) {
	defer wg.Done()

	for packet := range network.SendChannel {
		// logger.Debugf("received packet %v from channel", packet.Destination)
		bestNeighbour := utils.FindBestNeighbour(network.RoutingTable, packet)

		chord := pb.MiniChord{Message: &pb.MiniChord_NodeData{NodeData: packet}}

		// logger.Debugf("packet: s: %d | d: %d | sent to: %d", packet.Source, packet.Destination, bestNeighbour.Id)

		if packet.Source == node.Id {
			node.SendLock.Lock()
			// This packet originated at my node
			node.Stats.Sent++
			node.Stats.TotalSent += int64(packet.Payload)
			node.SendLock.Unlock()
		}

		err := utils.SendMessage(bestNeighbour.Connection, &chord)
		if err != nil {
			logger.Errorf("error forwarding packet to node %d: %s", bestNeighbour.Id, err.Error())
			os.Exit(1)
		}

		// VERY important sleep, as otherwise the network is overloaded.
		time.Sleep(1 * time.Millisecond)
	}
	// time.Sleep()
	logger.Infof("Finished sending packets.. closing connections")
	for _, peer := range network.RoutingTable {
		peer.Connection.Close()
	}
}
