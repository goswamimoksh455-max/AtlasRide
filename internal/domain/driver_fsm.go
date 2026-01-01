package domain

import "errors"

var (
	ErrInvalidTransition   = errors.New("invalid sriver state transition")
	ErrRideAlreadyAssigned = errors.New("Ride Already Assigned")
)

func CanTransition(from, to DriverStatus) bool {
	switch from {
	case DriverOffline:
		return to == DriverIdle
	case DriverIdle:
		return to == DriverMatching || to == DriverOffline
	case DriverMatching:
		return to == DriverBusy || to == DriverIdle
	case DriverBusy:
		return to == DriverIdle
	default:
		return false
	}
}
