package ride

import (
	"context"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type Repository interface {

	//Exactly-once ride creation
	//Enforced via DB uniqueness on rider_id.

	CreateIfAbsent(
		riderID string,
		driverID string,

	) (domain.Ride, bool, error)

	GetActiveByRider(
		ctx context.Context,
		riderID string,

	) (domain.Ride, bool, error)

	FindByID(
		riderID string,
	) (domain.Ride, error)
}
