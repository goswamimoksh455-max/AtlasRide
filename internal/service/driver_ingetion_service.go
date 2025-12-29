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
	now := time.Now()

	//loading existing driver if present
	driver, exists := s.repo.Get(driverID)

	if !exists {
		driver = domain.Driver{
			ID:     driverID,
			Status: domain.DriverIdle,
		}
	}

	//event-time assignment
	driver.Location = loc
	driver.UpdatedAt = now // event time

	// Repository decides accept / drop
	/*Monotonic Update means we only accept data if it is "newer" than what we already have.
	because of highCuncurrent system, over the network packets may arive in
	Out-Of-Order, so to ensure newer update is not accidental updated at by delayes pckt */
	accepted := s.repo.Upsert(driver)
	if !accepted {
		return
	}

	// Index mirrors accepted state ONLY
	s.index.Update(driver)
}
