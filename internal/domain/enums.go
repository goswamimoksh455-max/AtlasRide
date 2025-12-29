package domain

type DriverStatus int

// DriverStatus represents availability state.
// Invalid transitions should be prevented at service layer.
const (
	DriverIdle DriverStatus = iota
	DriverMatching
	DriverBusy
	DriverOffline
)

func (s DriverStatus) String() string {
	switch s {
	case DriverOffline:
		return "OFFLINE"
	case DriverIdle:
		return "IDLE"
	case DriverMatching:
		return "MATCHING"
	case DriverBusy:
		return "BUSY"
	default:
		return "UNKNOWN"
	}
}
