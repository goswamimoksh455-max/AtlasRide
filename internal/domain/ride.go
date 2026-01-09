package domain

import "time"

type RideStatus string

const (
	RideAssigned  RideStatus = "ASSIGNED"
	RideCanceled  RideStatus = "CANCELED"
	RideCompleted RideStatus = "COMPLETED"
)

type Ride struct {
	ID        string
	RiderID   string
	DriverID  string
	Status    RideStatus
	CreatedAt time.Time
}
