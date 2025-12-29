package service

import (
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
)

type DriverIngestionService struct {
	repo  repository.DriverRepository
	index spatial.SpatialIndex
}

func NewDriverIngetionService(
	repo repository.DriverRepository,
	index spatial.SpatialIndex,
	ttl time.Duration,
) *DriverIngestionService {
	return &DriverIngestionService{
		repo, index,
	}
}

func (s *DriverIngestionService) UpdateLocation(
	driverID string,
	loc domain.Location,
) {
	driver := domain.Driver{
		ID:        driverID,
		Location:  loc,
		Status:    domain.DriverIdle,
		UpdatedAt: time.Now(), // event time (best effort)
	}

	// Repository decides accept / drop
	/*Monotonic Update means we only accept data if it is "newer" than what we already have.
	because of highCuncurrent system, over the network packets may arive in
	Out-Of-Order, so to ensure newer update is not accidental updated at by delayes pckt */
	s.repo.Upsert(driver)

	// Index mirrors accepted state
	s.index.Update(driver)
}
