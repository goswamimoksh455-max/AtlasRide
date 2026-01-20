package ride

import (
	"context"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type RideService struct {
	repo Repository
}

func NewRideService(repo Repository) *RideService {
	return &RideService{repo: repo}
}

// finalizeAssignment is the ONLY way to rides are created.
func (s *RideService) FinalizeAssignment(
	ctx context.Context,
	riderID string,
	driverID string,
) (domain.Ride, bool, error) {
	return s.repo.CreateIfAbsent(ctx, riderID, driverID)
}

func (s *RideService) UpdateRideStatus(
	ctx context.Context,
	rideID string,
	newStatus domain.RideStatus,
) error {

	// 1. Load current ride (by rider or by ID later)
	ride, ok, err := s.repo.GetActiveByRider(ctx, rideID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrRideNotFound
	}

	// 2. Enforce invariant
	if err := domain.ValidateRideTransition(ride.Status, newStatus); err != nil {
		return err
	}

	// 3. Persist transition
	return s.repo.UpdateStatus(ctx, ride.ID, newStatus)
}
