package ride

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type MemoryRepo struct {
	mu            sync.Mutex
	rides         map[string]domain.Ride // keyed by riderID
	activeByRider map[string]string      // riderID → rideID
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		rides: make(map[string]domain.Ride),
	}
}

func (r *MemoryRepo) CreateIfAbsent(
	ctx context.Context,
	riderID string,
	driverID string,
) (domain.Ride, bool, error) {
	select {
	case <-ctx.Done():
		return domain.Ride{}, false, ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if ride, ok := r.rides[riderID]; ok {
		if ride.Status == domain.RideAssigned || ride.Status == domain.RideOngoing {
			return ride, false, nil
		}
	}

	now := time.Now()

	ride := domain.Ride{
		ID:        uuid.NewString(),
		RiderID:   riderID,
		DriverID:  driverID,
		Status:    domain.RideAssigned,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}

	r.rides[riderID] = ride
	return ride, true, nil
}

// func (r *MemoryRepo) FindByID(riderID string) (domain.Ride, error) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()

// 	ride, ok := r.rides[riderID]
// 	if !ok {
// 		return domain.Ride{}, errors.New("ride not found")
// 	}

// 	return ride, nil

// }

func (r *MemoryRepo) GetActiveByRider(
	ctx context.Context,
	riderID string,
) (domain.Ride, bool, error) {

	select {
	case <-ctx.Done():
		return domain.Ride{}, false, ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ride, ok := r.rides[riderID]
	if !ok {
		return domain.Ride{}, false, nil
	}

	if ride.Status != domain.RideAssigned && ride.Status != domain.RideOngoing {
		return domain.Ride{}, false, nil
	}

	return ride, true, nil
}

func (r *MemoryRepo) UpdateStatus(
	ctx context.Context,
	rideID string,
	status domain.RideStatus,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ride, ok := r.rides[rideID]
	if !ok {
		return errors.New("ride not found")
	}

	ride.Status = status
	ride.UpdatedAt = time.Now().UTC()

	r.rides[rideID] = ride
	return nil
}
