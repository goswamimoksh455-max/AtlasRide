package temp

import (
	"sync"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type OfferGroupStore struct {
	mu     sync.RWMutex
	groups map[string]*domain.OfferGroup //riderID -> group
}

func NewOfferGroupStore() *OfferGroupStore {
	return &OfferGroupStore{
		groups: make(map[string]*domain.OfferGroup),
	}
}

func (s *OfferGroupStore) Create(
	riderID string,
	driverIDs []string,
	ttl time.Duration,

) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.groups[riderID] = &domain.OfferGroup{
		RiderID:   riderID,
		DriverIDs: driverIDs,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (s *OfferGroupStore) Get(riderID string) (*domain.OfferGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	og, ok := s.groups[riderID]
	return og, ok
}

func (s *OfferGroupStore) Exists(riderID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.groups[riderID]
	return ok
}

func (s *OfferGroupStore) Remove(riderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.groups, riderID)
}

func (s *OfferGroupStore) OtherDrivers(riderID, driverID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.groups[riderID].DriverIDs
}

func (s *OfferGroupStore) WithLock(riderID string, fn func()) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.groups[riderID]
	if !ok {
		return
	}

	fn()

}

// /back-Ground event to clearUp
func (s *OfferGroupStore) CleanExpiredOffers() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiredMap := make(map[string][]string) // riderID -> []driverIDs
	now := time.Now()

	for riderID, group := range s.groups {
		if now.After(group.ExpiresAt) {
			expiredMap[riderID] = group.DriverIDs
			delete(s.groups, riderID)
		}
	}
	return expiredMap
}
