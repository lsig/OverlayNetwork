package registry

import (
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
)

func (r *Registry) HandleRegistration(conn net.Conn, msg *pb.MiniChord_Registration) {
	if r.SetupSent {
		logger.Error("Can't register after setup has been executed")
		return
	}

	var info string
	success := true

	registrationAddr := msg.Registration.GetAddress()

	if !verifyAddress(registrationAddr, conn.RemoteAddr().String()) {
		success = false
		info = "Registration request unsuccessful: Address mismatch."
	}

	if r.AddressExists(registrationAddr) {
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

func (r *Registry) HandleDeregistration(conn net.Conn, msg *pb.MiniChord_Deregistration) {
	var info string
	var id int32
	success := true

	if r.SetupSent {
		success = false
		info = "Can't Deregister after setup has executed"
	}

	registrationAddr := msg.Deregistration.GetAddress()

	if success && !verifyAddress(registrationAddr, conn.RemoteAddr().String()) {
		success = false
		info = "Deregistration request unsuccessful: Address mismatch."
	}

	if success && !r.AddressExists(registrationAddr) {
		success = false
		info = "Deregistration request unsuccessful: Address does not exist."
	}

	if success {
		info = fmt.Sprintf("Deregistration request successful. Node Id: (%d) not longer exists. The number of messaging nodes currently constituting the overlay is (%d).", id, len(r.Keys))
		id = r.RemoveNode(msg.Deregistration.GetId())
		logger.Info(info)
	} else {
		id = -1
		logger.Infof("node %d: %s", msg.Deregistration.Id, info)
	}

	res := &pb.DeregistrationResponse{
		Result: id,
		Info:   info,
	}

	chordMessage := &pb.MiniChord{
		Message: &pb.MiniChord_DeregistrationResponse{
			DeregistrationResponse: res,
		},
	}

	if err := r.SendMessage(conn, chordMessage); err != nil {
		errMsg := fmt.Sprintf("Failed to send deregistration response: %v", err)
		logger.Error(errMsg)

		if id != -1 {
			// Remove node if sending response fails
			r.AddNode(registrationAddr, conn)
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
	r.SetupSent = true
}

func (r *Registry) HandleNodeRegistryResponse(res *pb.MiniChord_NodeRegistryResponse) {
	if res.NodeRegistryResponse.Result > 127 {
		logger.Error("Node Failed to connect to Nodes in Routing table. Exiting ...")
		os.Exit(1)
	}
	r.NoSetupNodes++

	if r.NoSetupNodes == len(r.Keys) {
		logger.Info("The registry is now ready to initiate tasks.")
		r.SetupComplete = true
	}
}

func (r *Registry) HandleInitiateTask(task *pb.MiniChord) {
	for _, node := range r.Nodes {
		if err := r.SendMessage(node.Conn, task); err != nil {
			errMsg := fmt.Sprintf("Failed to send InitiateTask request: %v", err)
			logger.Error(errMsg)
		}
		logger.Info(fmt.Sprintf("Succesfully sent InitiateTask to node %d", node.Id))
	}
	r.StartComplete = true
}

func (r *Registry) HandleTaskFinished(conn net.Conn, msg *pb.MiniChord_TaskFinished) {
	if msg.TaskFinished.Id > 127 {
		logger.Error("Node Failed to Finish Sending Messages. Exiting ...")
		os.Exit(1)
	}
	if !verifyAddress(msg.TaskFinished.GetAddress(), conn.RemoteAddr().String()) {
		logger.Error("Node address does not match the connection address")
		return
	}

	r.NoFinished++

	if r.NoFinished == len(r.Keys) {
		// Sleep for 5 seconds to allow relaying packages to finish
		logger.Info("All packets arrived... sleeping 5 seconds")
		time.Sleep(5 * time.Second)
		r.sendTrafficReq()
	}
}

func (r *Registry) sendTrafficReq() {
	for _, node := range r.Nodes {
		req := &pb.RequestTrafficSummary{}
		chordMessage := &pb.MiniChord{
			Message: &pb.MiniChord_RequestTrafficSummary{
				RequestTrafficSummary: req,
			},
		}
		if err := r.SendMessage(node.Conn, chordMessage); err != nil {
			errMsg := fmt.Sprintf("Failed to send Traffic Request: %v", err)
			logger.Error(errMsg)
		}
		logger.Info(fmt.Sprintf("Succesfully sent Traffic Request to Node (%d) ", node.Id))
	}
}

func (r *Registry) HandleTrafficSummary(msg *pb.MiniChord_ReportTrafficSummary) {
	summary := Summary{
		Id:            msg.ReportTrafficSummary.GetId(),
		Sent:          msg.ReportTrafficSummary.GetSent(),
		Received:      msg.ReportTrafficSummary.GetReceived(),
		Relayed:       msg.ReportTrafficSummary.GetRelayed(),
		TotalSent:     msg.ReportTrafficSummary.GetTotalSent(),
		TotalReceived: msg.ReportTrafficSummary.GetTotalReceived(),
	}

	r.Summaries = append(r.Summaries, summary)

	if len(r.Keys) == len(r.Summaries) {
		r.printSummaries()
		r.Summaries = []Summary{}
		r.StartComplete = false
	}
}

func (r *Registry) printSummaries() {
	var sentSum, receivedSum uint32
	var totalSentSum, totalReceivedSum int64
	for _, s := range r.Summaries {
		fmt.Printf("Node %d,%d,%d,%d,%d,%d\n",
			s.Id,
			s.Sent,
			s.Received,
			s.Relayed,
			s.TotalSent,
			s.TotalReceived,
		)
		sentSum += s.Sent
		receivedSum += s.Received
		totalSentSum += s.TotalSent
		totalReceivedSum += s.TotalReceived
	}
	fmt.Printf("Total | %d, %d, %d, %d\n", sentSum, receivedSum, totalSentSum, totalReceivedSum)
}

// Command Line Handlers

func (r *Registry) HandleSetup(routingTableSize int) {
	if r.SetupSent {
		logger.Error("Setup already complete")
		return
	}

	maxSize := math.Floor(math.Log2(float64(len(r.Nodes))))

	if routingTableSize > int(maxSize) {
		logger.Warning(fmt.Sprintf("Routing table size %d too large for network size %d. Setting size to maximum: %d", routingTableSize, len(r.Keys), int(maxSize)))
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

func (r *Registry) HandleStart(nopackets int) {
	if !r.SetupComplete {
		logger.Error("Setup not complete")
		return
	}

	if r.StartComplete {
		logger.Error("Start already completed")
		return
	}

	if nopackets < 1 {
		logger.Error("Number of packets must be positive")
		return
	}

	start := &pb.InitiateTask{
		Packets: uint32(nopackets),
	}

	miniChordMsg := &pb.MiniChord{
		Message: &pb.MiniChord_InitiateTask{
			InitiateTask: start,
		},
	}

	msg := &Packet{
		Conn:    nil,
		Content: miniChordMsg,
	}

	r.Packets <- msg
}

func (r *Registry) HandleList() {
	if len(r.Keys) == 0 {
		logger.Error("No node is connected to the registry")
		return
	}
	fmt.Println("Node IDs and Addresses:")
	fmt.Println("-----------------------")

	for _, node := range r.Nodes {
		fmt.Printf("ID: %d, Address: %s\n", node.Id, node.Address)
	}
}

func (r *Registry) HandleRouteCmd() {
	if !r.SetupSent {
		logger.Error("Setup not complete, routing tables have not been calculated")
		return
	}
	for _, node := range r.Nodes {
		fmt.Printf("Routing Table for Node %d:\n", node.Id)
		fmt.Println("Node ID\tAddress")
		fmt.Println("-------\t-------")

		for id, address := range node.RoutingTable {
			fmt.Printf("%d\t%s\n", id, address)
		}
		fmt.Println()
	}

}
