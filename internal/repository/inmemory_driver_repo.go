package repository

//repo itself is the final gateKeeper
import (
	"errors"
	"fmt"
	"log/slog"
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
		slog.Info(fmt.Sprintf(" Droped Late Packet, Driver_id:%d", incoming.ID))
		return false
		//due to pckt delays , current packet for update may be got delayed in the Network Hiccups ":)"
	}

	//drop future timestamps (clock skew protection)
	//Allow small skew (e.g., GPS jitters)
	const maxSkew = 2 * time.Second
	if incoming.UpdatedAt.After(now.Add(maxSkew)) {
		slog.Info(fmt.Sprintf("Droped large skew, for clock skew protection , id:%d", incoming.ID))
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

func (r *InMemoryDriverRepository) TransitionStatus(
	driverID string,
	to domain.DriverStatus,
) error {

	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.drivers[driverID]
	if !ok {
		slog.Error("Driver Not Found")
		return errors.New("driver not found")
	}

	from := entry.driver.Status

	if !domain.CanTransition(from, to) {
		slog.Error("Error - invalid Transition")
		return domain.ErrInvalidTransition
	}

	now := time.Now()

	//FSM side-effects
	switch to {
	case domain.DriverMatching:
		entry.driver.MatchingSince = &now

	case domain.DriverBusy:
		entry.driver.MatchingSince = nil
		entry.driver.Intent = nil //critical cleanup

	case domain.DriverIdle:
		entry.driver.MatchingSince = nil
		entry.driver.Intent = nil // safe cleanup
	}

	entry.driver.Status = to
	entry.driver.UpdatedAt = time.Now()

	r.drivers[driverID] = entry

	return nil
}

func (r *InMemoryDriverRepository) FilterByStatus(
	ids []string,
	status domain.DriverStatus,
) []string {

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0)
	for _, id := range ids {
		entry, ok := r.drivers[id]
		if ok && entry.driver.Status == status {
			result = append(result, id)
		}
	}

	return result
}

// Sweeper for Scan Drivers and Who are too long in the
// Matching state due to any resone Roll them back to IDLE
func (r *InMemoryDriverRepository) FindStuckMatching(
	timeout time.Duration,
) []string {

	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	var stuck []string

	for id, entry := range r.drivers {
		if (entry.driver.Status == domain.DriverMatching &&
			entry.driver.MatchingSince != nil &&
			now.Sub(*entry.driver.MatchingSince) > timeout) ||
			(entry.driver.HasActiveIntent(now) && now.Sub(entry.driver.Intent.ExpiresAt) > timeout) ||
			(now.After(entry.driver.Intent.ExpiresAt)) {

			stuck = append(stuck, id)
		}
	}

	return stuck
}

// Adding Intent Lock Method
func (r *InMemoryDriverRepository) TryLockIntent(
	driverID string,
	riderID string,
	ttl time.Duration,

) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.drivers[driverID]
	if !ok {
		return errors.New("Driver Not Found")
	}

	now := time.Now()

	//in case somene else already locked it
	if entry.driver.HasActiveIntent(now) {
		return errors.New("intent already locked")
	}

	entry.driver.Intent = &domain.MatchIntent{
		RiderID:   riderID,
		ExpiresAt: now.Add(ttl),
	}
	entry.driver.UpdatedAt = now

	r.drivers[driverID] = entry
	return nil
}

//cuncurrency gate

func (r *InMemoryDriverRepository) ClearIntent(driverID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.drivers[driverID]
	if !ok {
		return
	}

	entry.driver.Intent = nil
	entry.driver.UpdatedAt = time.Now()
	r.drivers[driverID] = entry
}

func (r *InMemoryDriverRepository) RespondToIntent(
	driverID string,
	riderID string,
	response domain.DriverResponse,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.drivers[driverID]
	if !ok {
		return errors.New("Driver not Found")
	}

	d := entry.driver

	if d.Status != domain.DriverMatching ||
		d.Intent == nil ||
		d.Intent.RiderID != riderID {
		return errors.New("Invalid Intent")
	}

	switch response {
	case domain.Accept:
		d.Status = domain.DriverBusy
		d.Intent = nil
		d.MatchingSince = nil

	case domain.Reject:
		d.Status = domain.DriverIdle
		d.Intent = nil
		d.MatchingSince = nil
	}

	entry.driver = d
	r.drivers[driverID] = entry
	return nil
}
