package spatial

import "github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"

// SpatialIndex provides fast geo-based lookup.
// Implementations must be concurrency-safe.
type SpatialIndex interface {
	Insert(driver domain.Driver) error
	Remove(driverId string)
	Update(driver domain.Driver)
	Nearby(cellId string, k int) []domain.Driver
}
