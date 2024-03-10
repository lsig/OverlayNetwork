package registry

import (
	"fmt"
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

	id := r.AddNode(registrationAddr)

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

func verifyAddress(clientAddr string, connAddr string) bool {
	clientIp, _, err := net.SplitHostPort(clientAddr)

	if err != nil {
		return false
	}

	connIp, _, err := net.SplitHostPort(connAddr)

	if err != nil {
		return false
	}

	if clientIp == connIp {
		return true
	}

	return false
}
