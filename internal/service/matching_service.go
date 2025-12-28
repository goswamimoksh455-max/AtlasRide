package service

import (
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/matching"
)

// MatchingService orchestrates matching use-cases.
type MatchingService struct {
	matcher matching.Matcher
}

func NewMatchingService(m matching.Matcher) *MatchingService {
	return &MatchingService{matcher: m}
}

func (s *MatchingService) Match(rider domain.Rider) (*domain.Driver, error) {
	return s.matcher.FindNearestDriver(rider)
}
