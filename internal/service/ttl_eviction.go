package service

import (
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
)

type TTLEvictionService struct {
	repo  repository.DriverRepository
	index spatial.SpatialIndex
	ttl   time.Duration
}

func NewTTLEvictionService(
	repo repository.DriverRepository,
	index spatial.SpatialIndex,
	ttl time.Duration,
) *TTLEvictionService {
	return &TTLEvictionService{repo, index, ttl}
}

func (s *TTLEvictionService) RunOnce() {
	expired := s.repo.Expired(s.ttl)
	for _, id := range expired {
		s.repo.Delete(id)
		s.index.Remove(id)
	}

}
