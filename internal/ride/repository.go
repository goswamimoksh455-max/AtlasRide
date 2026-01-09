package ride

import "github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"

type Repository interface {

	//Exactly-once ride creation
	//Enforced via DB uniqueness on rider_id.

	CreateIfAbsent(
		riderID string,
		driverID string,

	) (domain.Ride, bool, error)

	FindByID(
		riderId string,

	) (domain.Ride, error)
}
