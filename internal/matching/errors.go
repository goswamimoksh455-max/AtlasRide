package matching

import "errors"

var (
	ErrNoDriversAvailable = errors.New("No Drivers Available")
	ErrDriverAlreadyTaken = errors.New("Driver Already Assigned")
)
