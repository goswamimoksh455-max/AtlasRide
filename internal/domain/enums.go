package domain

type DriverStatus int

// DriverStatus represents availability state.
// Invalid transitions should be prevented at service layer.
const (
	DriverIdle DriverStatus = iota
	DriverMatching
	DriverBusy
)
