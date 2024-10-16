package main

import (
	"os"

	"github.com/lsig/OverlayNetwork/logger"
	"github.com/lsig/OverlayNetwork/registry/registry"
)

func main() {
	r, err := registry.NewRegistry("8080")

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	go r.Start()
	go r.CommandLineInterface()

	select {}
}
