package ride

import "github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"

type RideService struct {
	repo Repository
}

func NewRideService(repo Repository) *RideService {
	return &RideService{repo: repo}
}

// finalizeAssignment is the ONLY way to rides are created.
func (s *RideService) FinalizeAssignment(
	riderID string,
	driverID string,
) (domain.Ride, bool, error) {
	return s.repo.CreateIfAbsent(riderID, driverID)
}
