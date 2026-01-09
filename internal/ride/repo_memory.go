package ride

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type MemoryRepo struct {
	mu    sync.Mutex
	rides map[string]domain.Ride //keyed by riderID

}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		rides: map[string]domain.Ride{},
	}
}

func (r *MemoryRepo) CreateIfAbsent(
	riderID string,
	driverID string,

) (domain.Ride, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ride, ok := r.rides[riderID]; ok {
		return ride, false, nil
	}

	ride := domain.Ride{
		ID:        uuid.NewString(),
		RiderID:   riderID,
		DriverID:  driverID,
		Status:    domain.RideAssigned,
		CreatedAt: time.Now(),
	}

	r.rides[riderID] = ride

	return ride, true, nil
}

func (r *MemoryRepo) FindByID(riderID string) (domain.Ride, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ride, ok := r.rides[riderID]
	if !ok {
		return domain.Ride{}, errors.New("ride not found")
	}

	return ride, nil

}
