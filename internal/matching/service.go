package matching

import (
	"sort"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
	"github.com/uber/h3-go/v4"
)

//interface Segregation , i think like this : interface = contract, struct = worker

type DriverStore interface {
	Get(id string) (domain.Driver, bool)
	Upsert(domain.Driver) bool
	TransitionStatus(id string, to domain.DriverStatus) error
}

type SpatialIndex interface {
	Nearby(cell h3.Cell, k int) []domain.Driver
	CellIDForLocation(loc domain.Location) (h3.Cell, error)
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
		// reserve driver (exclusive lock via FSM)
		err := s.drivers.TransitionStatus(
			c.driverID,
			domain.DriverMatching,
		)
		if err != nil {
			continue //race lost
		}

		//commit
		err = s.drivers.TransitionStatus(
			c.driverID,
			domain.DriverBusy,
		)
		if err == nil {
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
