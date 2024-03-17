package main

import (
	"os"
	"sync"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/messages/helpers"
	"github.com/lsig/OverlayNetwork/messages/utils"
)

func main() {
	registry, err := utils.GetRegistryFromProgramArgs(os.Args)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// create Listener Node
	node, err := helpers.CreateListenerNode()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer node.Listener.Close()

	// Connect to registry
	if err = helpers.ConnectToRegistry(registry); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// send Registration
	registrationResponse, err := helpers.Register(node, registry)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	node.Id = registrationResponse.Result

	// wait for Node Registry
	nodeRegistry, err := helpers.GetNodeRegistry(registry)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.Debugf("Ids: %v", nodeRegistry.Ids)

	// setup network
	network, err := helpers.SetupNetwork(nodeRegistry, node)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	logger.Debug("RoutingTable: ")
	for _, peer := range network.RoutingTable {
		logger.Debugf("node: %d - address %s", peer.Id, peer.Address.ToString())
	}

	wg := sync.WaitGroup{}
	wg.Add(4)

	go helpers.HandleListener(&wg, node, network)
	helpers.ConnectToNeighbours(network)

	// Send NodeRegistry Response
	if err = helpers.SendNodeRegistryResponse(node, network, registry); err != nil {
		logger.Errorf("error sending NodeRegistryResponse to registry: %s", err.Error())
		os.Exit(1)
	}

	packets, err := helpers.GetInitiateTasks(registry)
	if err != nil {
		logger.Errorf("error receiving Initiate Tasks: %s", err.Error())
		os.Exit(1)
	}

	// create and add packets to sendChannel
	go helpers.CreatePackets(node, network, packets)

	// go HandleRegistry(&wg, registry)
	go helpers.HandleConnector(&wg, network)
	go helpers.HandleStdInput(&wg, node, registry)
	wg.Wait()

	logger.Info("I'm done now... bye")
}
