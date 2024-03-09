package registry

import (
	"fmt"
	"net"

	"github.com/lsig/OverlayNetwork/logger"
	pb "github.com/lsig/OverlayNetwork/pb"
)

func (r *Registry) HandleRegistration(conn net.Conn, msg *pb.MiniChord_Registration) {
	address := conn.RemoteAddr().String()
	var result int32 = -1
	var info string

	if address != msg.Registration.GetAddress() {
		info = "Registration request unsuccessful: Address mismatch."
		logger.Error(info)
	} else {
		id := r.AddNode(address)
		if id != -1 {
			result = id
			info = fmt.Sprintf("Registration request successful. The number of messaging nodes currently constituting the overlay is (%d).", len(r.Keys))
		} else {
			info = "Registration request unsuccessful."
			logger.Error(info)
		}
	}

	res := &pb.RegistrationResponse{
		Result: result,
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

		if result != -1 {
			// Remove node if sending response fails
			r.RemoveNode(result)
		}
	}

	logger.Info(info)
}
