package domain

import "time"

type OfferGroup struct {
	RiderID   string
	DriverIDs []string
	ExpiresAt time.Time
}
