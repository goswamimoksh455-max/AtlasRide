package domain

import (
	"time"
)

type MatchIntent struct {
	RiderID   string
	ExpiresAt time.Time
}
type Driver struct {
	ID        string
	Location  Location
	Status    DriverStatus
	UpdatedAt time.Time

	MatchingSince *time.Time   //nil unless matching
	Intent        *MatchIntent //temp reservation - Soft lock
}

func (d Driver) HasActiveIntent(now time.Time) bool {
	if d.Intent == nil {
		return false
	}
	return now.Before(d.Intent.ExpiresAt) //T:intent is not expired so Driver not available
}
