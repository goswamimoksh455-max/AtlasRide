package domain

import (
	"errors"
	"time"
)

// RideStatus represents the lifecycle of a ride.
// A ride is ACTIVE if status ∈ {assigned, ongoing}
type RideStatus string

var ErrRideNotFound = errors.New("Rider Not Found")

const (
	RideAssigned  RideStatus = "assigned"  // driver accepted, not yet started
	RideOngoing   RideStatus = "ongoing"   // rider picked up
	RideCompleted RideStatus = "completed" // ride finished
	RideCanceled  RideStatus = "canceled"  // rider or driver canceled
)

// Ride is the durable source of truth for an assignment.
// A ride MUST exist before a driver transitions to BUSY.
type Ride struct {
	ID        string
	RiderID   string
	DriverID  string
	Status    RideStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
