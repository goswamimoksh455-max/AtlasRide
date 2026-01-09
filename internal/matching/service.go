package matching

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/events"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/offer"
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
	//TryLockIntent(driverID string, riderID string, ttl time.Duration) error
	RespondToIntent(driverID string, riderID string, response domain.DriverResponse) error
	SetIntent(driverID string, riderID string) error
}

type SpatialIndex interface {
	Nearby(cell h3.Cell, k int) []domain.Driver
	CellIDForLocation(loc domain.Location) (h3.Cell, error)
}

type RideFinalizer interface {
	FinalizeAssignment(riderID, driverID string) (domain.Ride, bool, error)
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
	offerGroups offer.RedisOfferGroupStore
	rides       RideFinalizer
}

func NewService(drivers DriverStore, spatial SpatialIndex, offerGroups offer.RedisOfferGroupStore, rides RideFinalizer) *Service {
	return &Service{
		drivers:     drivers,
		spatial:     spatial,
		offerGroups: offerGroups,
		rides:       rides,
	}
}

func (s *Service) SetEventDispatcher(dispatcher events.Dispatcher) {
	s.events = dispatcher
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

		//redis provides first-accept-wins
		//below code is no longer needed
		// err := s.drivers.TryLockIntent(
		// 	c.driverID,
		// 	req.RiderID,
		// 	5*time.Second,
		// )
		// if err != nil {
		// 	continue //somene else won in cuncrrent attempt
		// }

		// reserve driver (exclusive lock via FSM)
		// Move driver into matching state
		err = s.drivers.TransitionStatus(
			c.driverID,
			domain.DriverMatching,
		)
		if err != nil {
			continue
		}

		// Attach intent (THIS WAS MISSING)
		err = s.drivers.SetIntent(
			c.driverID,
			req.RiderID,
		)
		if err != nil {
			_ = s.drivers.TransitionStatus(
				c.driverID,
				domain.DriverIdle,
			)
			continue
		}

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
	//redis already hold the intent locks so no need of them after switching from the inmenmory
	err = s.offerGroups.Create(
		context.Background(),
		req.RiderID,
		5, //seconds
	)
	if err != nil {
		//You must rollback intent if OfferGroup creation fails.
		for _, d := range lockedDrivers {
			s.drivers.ClearIntent(d)
			_ = s.drivers.TransitionStatus(
				d,
				domain.DriverIdle,
			)
		}
		return nil, err
	}

	//Emiting offers asynchronously, but maintaining control over the GO routine
	//but need to control the Go routine count
	//note : Backpressure means: “If downstream can’t keep up, upstream must slow down.”

	for _, driverID := range lockedDrivers {
		err := s.events.EnqueueDriverOffer(driverID, req.RiderID)
		if err != nil {
			slog.Warn("offer queue full",
				"driver", driverID,
				"rider", req.RiderID,
			)
		}
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

	ctx := context.Background()

	// --- Reject path ---
	if response == domain.Reject {
		slog.Info("[REJECT]", "driver", driverID, "rider", riderID)

		s.drivers.ClearIntent(driverID)
		_ = s.drivers.TransitionStatus(driverID, domain.DriverIdle)

		return nil
	}

	// --- ACCEPT path (atomic in Redis) ---
	won, err := s.offerGroups.Accept(ctx, riderID, driverID)
	if err != nil {
		// Check if error is because offer group doesn't exist
		// This is actually OK - it means someone already won and removed it
		if err.Error() == "offer expired" {
			slog.Info("[ACCEPT_AFTER_WINNER]",
				"driver", driverID,
				"rider", riderID,
			)

			// Just clean up this driver silently
			s.drivers.ClearIntent(driverID)
			_ = s.drivers.TransitionStatus(driverID, domain.DriverIdle)

			return nil // Not an error - race condition is expected
		}

		// Real error - revert driver
		slog.Warn("[ACCEPT_ERROR]",
			"driver", driverID,
			"error", err,
		)

		s.drivers.ClearIntent(driverID)
		_ = s.drivers.TransitionStatus(driverID, domain.DriverIdle)

		return err
	}

	if !won {
		// Lost the race → revert driver to idle
		slog.Info("[ACCEPT_LOST]", "driver", driverID, "rider", riderID)

		s.drivers.ClearIntent(driverID)
		_ = s.drivers.TransitionStatus(driverID, domain.DriverIdle)

		return nil
	}

	//Persist Outcome (durable)
	// 	Becomes:
	// HTTP call or
	// gRPC call or
	// Kafka command
	ride, created, err := s.rides.FinalizeAssignment(riderID, driverID)
	if err != nil {
		slog.Error("RIDE_FINALIZE_ERROR",
			"driver", driverID,
			"rider", riderID,
			"error", err,
		)

		//revert driver intent
		s.drivers.ClearIntent(driverID)
		_ = s.drivers.TransitionStatus(driverID, domain.DriverIdle)

		return err
	}

	if !created {
		//Someone else already finalized
		slog.Info("RIDE_ALREADY_EXISTS",
			"driver", driverID,
			"rider", riderID,
			"ride", ride.ID,
		)

		s.drivers.ClearIntent(driverID)
		_ = s.drivers.TransitionStatus(driverID, domain.DriverIdle)

		return nil
	}

	// --- WINNER path ---
	err = s.drivers.TransitionStatus(driverID, domain.DriverBusy)
	if err != nil {
		slog.Error("[ACCEPT_WINNER] transition failed",
			"driver", driverID,
			"error", err,
		)
		return err
	}

	slog.Info("[ACCEPT_WINNER]", "driver", driverID, "rider", riderID)

	// DON'T remove offer group here - let it expire naturally
	// This allows late acceptors to see the group still exists
	// Redis TTL will clean it up after 5 seconds

	return nil
}
