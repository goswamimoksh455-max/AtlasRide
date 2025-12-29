package repository

//repo itself is the final gateKeeper
import (
	"sync"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type driverEntry struct {
	driver     domain.Driver
	lastSeenAt time.Time //server time (TTL only)
}
type InMemoryDriverRepository struct {
	mu      sync.RWMutex
	drivers map[string]driverEntry
}

func NewInMemoryDriverRepository() *InMemoryDriverRepository {
	return &InMemoryDriverRepository{
		drivers: make(map[string]driverEntry),
	}
}

func (r *InMemoryDriverRepository) Upsert(incoming domain.Driver) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	existing, exists := r.drivers[incoming.ID]

	//drop late packets ( event-time monotonicity)
	if exists && incoming.UpdatedAt.Before(existing.driver.UpdatedAt) {
		return false
		//due to pckt delays , current packet for update may be got delayed in the Network Hiccups ":)"
	}

	//drop future timestamps (clock skew protection)
	//Allow small skew (e.g., GPS jitters)
	const maxSkew = 2 * time.Second
	if incoming.UpdatedAt.After(now.Add(maxSkew)) {
		return false
	}

	r.drivers[incoming.ID] = driverEntry{
		driver:     incoming,
		lastSeenAt: now, //server time for TTL
	}

	return true
}

func (r *InMemoryDriverRepository) Get(id string) (domain.Driver, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.drivers[id]
	return entry.driver, ok
}

func (r *InMemoryDriverRepository) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.drivers, id) //amortised O(1)
}

func (r *InMemoryDriverRepository) Expired(ttl time.Duration) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var expired []string

	for id, entry := range r.drivers {
		if now.Sub(entry.lastSeenAt) > ttl {
			expired = append(expired, id)
		}
	}
	return expired
}

func (r *InMemoryDriverRepository) All() []domain.Driver {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Driver, 0, len(r.drivers))
	for _, d := range r.drivers {
		result = append(result, d.driver)
	}
	return result
	//direct return of map , means returning shallow copy, prone to race condition
	/*By copying the data into a slice ([]domain.Driver), you are creating a
	"Point-in-Time Snapshot." Once the slice is created and returned,
	the caller can read it safely without needing to hold the lock,
	and the Repository can continue updating the original map*/
}

//IMP to keep the repo swapable with the Redis latter we
//are not keeping the TTL logic in the REPO so it dont become time-aware
