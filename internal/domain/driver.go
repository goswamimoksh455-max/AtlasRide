package domain

import (
	"time"
)

type Driver struct {
	ID        string
	Location  Location
	Status    DriverStatus
	UpdatedAt time.Time

	MatchingSince *time.Time //nil unless matching
}
