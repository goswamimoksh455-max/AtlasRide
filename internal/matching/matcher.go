package matching

import "github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"

// Matcher encapsulates deterministic matching logic.
type Matcher interface {
	FindNearestDriver(rider domain.Rider) (*domain.Driver, error)
}
