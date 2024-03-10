package utils

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"

	"github.com/lsig/OverlayNetwork/messages/types"
)

func GetRegistryFromProgramArgs(args []string) (*types.Registry, error) {
	usageError := fmt.Errorf("usage: go run messages/messages.go <registry-host>:<registry-port>")
	if len(args) != 2 {
		return nil, usageError
	}

	addressInfo := strings.Split(args[1], ":")
	if len(addressInfo) != 2 {
		return nil, usageError
	}

	if addressInfo[0] == "localhost" {
		addressInfo[0] = "127.0.0.1"
	}

	address := net.ParseIP(addressInfo[0])
	port, err := strconv.Atoi(addressInfo[1])

	if address == nil || err != nil || port <= 0 || port >= 65536 {
		return nil, usageError
	}

	registry := types.Registry{Address: address, Port: uint16(port)}

	return &registry, nil
}

func GenerateRandomPort() int {
	randomPort := -1

	for randomPort < 0 {
		randomPort = rand.Intn(int(math.Pow(2, 16)))
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", randomPort))
		if err == nil {
			fmt.Printf("server live on: %v\n", conn.RemoteAddr().String())
			randomPort = -1
		} else {
			conn.Close()
		}
	}

	return randomPort
}
