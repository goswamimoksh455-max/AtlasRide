package ride

import (
	"context"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type Repository interface {

	//Exactly-once ride creation
	//Enforced via DB uniqueness on rider_id.

	CreateIfAbsent(
		ctx context.Context,
		riderID string,
		driverID string,
	) (domain.Ride, bool, error)

	GetActiveByRider(
		ctx context.Context,
		riderID string,

	) (domain.Ride, bool, error)

	UpdateStatus(
		ctx context.Context,
		rideID string,
		status domain.RideStatus,
	) error

	// FindByID(
	// 	riderID string,
	// ) (domain.Ride, error)
}
