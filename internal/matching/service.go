package matching

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/events"
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
	RespondToIntent(driverID string, riderID string, response domain.DriverResponse) error
}

type SpatialIndex interface {
	Nearby(cell h3.Cell, k int) []domain.Driver
	CellIDForLocation(loc domain.Location) (h3.Cell, error)
}

type OfferGroupStore interface {
	Create(riderID string, driverIDs []string, ttl time.Duration)
	Exists(riderID string) bool
	Remove(riderID string)
	OtherDrivers(riderID, driverID string) []string
	WithLock(riderID string, fn func())
}

// added for Recovery from Too long Matching stuck
type RecoveryService struct {
	drivers DriverStore
	timeout time.Duration
}
type Service struct {
	drivers     DriverStore
	spatial     SpatialIndex
	events      events.Dispatcher
	offerGroups OfferGroupStore
}

func NewService(drivers DriverStore, spatial SpatialIndex, events events.Dispatcher, offerGroups OfferGroupStore) *Service {
	return &Service{
		drivers:     drivers,
		spatial:     spatial,
		events:      events,
		offerGroups: offerGroups,
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

	fanout := 3
	if len(candidateRanked) < fanout {
		fanout = len(candidateRanked)
	}

	lockedDrivers := make([]string, 0, fanout)

	for i := 0; i < fanout; i++ {

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

		//emmiting Offer (async boundary)
		// err = s.drivers.TransitionStatus( //bussy statu clear the intent in transition
		// 	c.driverID,
		// 	domain.DriverBusy,
		// )
		lockedDrivers = append(lockedDrivers, c.driverID)

		slog.Info(fmt.Sprintf("[MATCH] rider=%s trying driver=%s dist=%.2fm\n", req.RiderID, c.driverID, c.distance))
		// return &MatchResult{
		// 	RiderID:  req.RiderID,
		// 	DriverID: c.driverID,
		// 	Distance: c.distance,
		// }, nil

		// 		_ = s.drivers.TransitionStatus(
		// 			c.driverID,
		// 			domain.DriverIdle,
		// 		)

	}
	if len(lockedDrivers) == 0 {
		return nil, ErrNoDriversAvailable
	}

	//creating offer group (first-accept-wins)
	s.offerGroups.Create(
		req.RiderID,
		lockedDrivers,
		5*time.Second,
	)

	//Emiting offers asynchronously
	for _, driverID := range lockedDrivers {
		go s.events.EmitDriverOffer(driverID, req.RiderID)
	}

	return &MatchResult{
		RiderID:  req.RiderID,
		DriverID: "", //assigned asynchronously
	}, nil
}

//no waiting for Driver response
//NO busy Transition here

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

func (s *Service) HandleDriverResponse(
	driverID string,
	riderID string,
	response domain.DriverResponse,
) error {

	var resultErr error
	s.offerGroups.WithLock(riderID, func() {
		//critical section starts , HERE we go ..

		//offer already revolved ?
		if !s.offerGroups.Exists(riderID) {
			resultErr = domain.ErrRideAlreadyAssigned
			return
		}

		if response == domain.Accept {
			//winner
			err := s.drivers.RespondToIntent(driverID, riderID, domain.Accept)
			if err != nil {
				resultErr = err
				return
			}

			//cancel others
			others := s.offerGroups.OtherDrivers(riderID, driverID)
			for _, otherID := range others {
				_ = s.drivers.RespondToIntent(otherID, riderID, domain.Reject)
			}
			//remove Rider Request
			s.offerGroups.Remove(riderID)

			//******TODO*******
			//- Persist assignment
			//-Notify Rider

			resultErr = nil
			return
		} else {
			//in case of reject
			resultErr = s.drivers.RespondToIntent(driverID, riderID, domain.Reject)
			return
		}

	})

	return resultErr
}

//Note : Closure Power: In Go, anonymous functions can "see" and modify variables in their parent scope (like resultErr).
