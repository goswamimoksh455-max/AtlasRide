package matching

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
	"github.com/uber/h3-go/v4"
)

//interface Segregation , i think like this : interface = contract, struct = worker

type DriverStore interface {
	Get(id string) (domain.Driver, bool)
	Upsert(domain.Driver) bool
	TransitionStatus(id string, to domain.DriverStatus) error
	FindStuckMatching(timeout time.Duration) []string //stuck driver IDs
	ClearIntent(driverID string)
	TryLockIntent(driverID string, riderID string, ttl time.Duration) error
}

type SpatialIndex interface {
	Nearby(cell h3.Cell, k int) []domain.Driver
	CellIDForLocation(loc domain.Location) (h3.Cell, error)
}

// added for Recovery from Too long Matching stuck
type RecoveryService struct {
	drivers DriverStore
	timeout time.Duration
}
type Service struct {
	drivers DriverStore
	spatial SpatialIndex
}

func NewService(drivers DriverStore, spatial SpatialIndex) *Service {
	return &Service{
		drivers: drivers,
		spatial: spatial,
	}
}

func (s *Service) Match(req MatchRequest) (*MatchResult, error) {

	//Compute Rider H3 cell
	cell, err := s.spatial.CellIDForLocation(domain.Location{
		Lat: req.Lat,
		Lng: req.Lng,
	})
	if err != nil {
		return nil, err
	}

	//then Get nearby IDLE drivers (snapshot)
	candidates := s.spatial.Nearby(cell, req.K)

	//distance filter + best pick
	//finding closest
	candidateRanked := make([]candidate, 0, len(candidates))

	for _, d := range candidates {
		dist := spatial.DistanceMeters(
			req.Lat, req.Lng,
			d.Location.Lat,
			d.Location.Lng,
		)

		if dist > req.MaxDist {
			continue
		}

		candidateRanked = append(candidateRanked, candidate{
			driverID: d.ID,
			distance: dist,
		})
	}

	sort.Slice(candidateRanked, func(i, j int) bool {
		return candidateRanked[i].distance < candidateRanked[j].distance
	})

	maxAttempts := 3
	if len(candidateRanked) < maxAttempts {
		maxAttempts = len(candidateRanked)
	}

	for i := 0; i < maxAttempts; i++ {

		c := candidateRanked[i]

		//Re-Validate against source of truth
		driver, ok := s.drivers.Get(c.driverID)
		if !ok || driver.Status != domain.DriverIdle {
			continue
		}

		//trying to aquire intent lock ( cuncurrency real gate)
		err := s.drivers.TryLockIntent(
			c.driverID,
			req.RiderID,
			5*time.Second,
		)
		if err != nil {
			continue //somene else won in cuncrrent attempt
		}

		// reserve driver (exclusive lock via FSM)
		err = s.drivers.TransitionStatus(
			c.driverID,
			domain.DriverMatching,
		)
		if err != nil {
			s.drivers.ClearIntent(c.driverID) //rollback intent in FSM fails
			continue                          //race lost
		}

		//commit needed to be done by the Driver,then bottom code will execute

		err = s.drivers.TransitionStatus( //bussy statu clear the intent in transition
			c.driverID,
			domain.DriverBusy,
		)
		if err == nil {
			slog.Info(fmt.Sprintf("[MATCH] rider=%s trying driver=%s dist=%.2fm\n", req.RiderID, c.driverID, c.distance))
			return &MatchResult{
				RiderID:  req.RiderID,
				DriverID: c.driverID,
				Distance: c.distance,
			}, nil
		} else {
			_ = s.drivers.TransitionStatus(
				c.driverID,
				domain.DriverIdle,
			)
		}

	}

	return nil, ErrNoDriversAvailable

}

func NewRecoveryService(drivers DriverStore) *RecoveryService {
	return &RecoveryService{
		drivers: drivers,
		timeout: 5 * time.Second,
	}
	//just nneed to pass the DriverStore instance to make the RecoverService
	//DriverStore is just to limit the Interaction Methods of the InMemoryDriverService
}

func (s *RecoveryService) Recover() {

	slog.Info("recovery tick")
	stuckDrivers := s.drivers.FindStuckMatching(s.timeout)

	for _, driverID := range stuckDrivers {
		slog.Info(fmt.Sprintf("driver was stuck, id:%s", driverID))

		_ = s.drivers.TransitionStatus(
			driverID,
			domain.DriverIdle,
		)

		s.drivers.ClearIntent(driverID)
	}

	//intent is concurrency lock
}

//Idempotent
//safe to run Repeatedly
//No coordination Required
