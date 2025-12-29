package spatial

//29-12-2025 - Corrected under churn, race, TTL, reordering
import (
	"math"
	"sync"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/uber/h3-go/v4"
)

// type H3Index struct {
// 	resolution    int
// 	cellToDrivers map[h3.Cell][]string
// 	driverToCell  map[string]h3.Cell
// 	mu            sync.RWMutex
// }

// this new vorsion is good, stores the snap-shot in-memory
// for deterministic latency , and no more dependency on repository during matching
type H3Index struct {
	resolution    int
	cellToDrivers map[h3.Cell]map[string]domain.Driver
	driverToCell  map[string]h3.Cell
	mu            sync.RWMutex
	//slightly more memory but safer, and O(1) Delete, Cleaner Updates
}

func NewH3Index(reslution int) *H3Index {
	return &H3Index{
		resolution:    reslution,
		cellToDrivers: make(map[h3.Cell]map[string]domain.Driver),
		driverToCell:  make(map[string]h3.Cell),
	}
}

func (h *H3Index) cellForLocation(loc domain.Location) (h3.Cell, error) {
	cell, err := h3.LatLngToCell(h3.LatLng{
		Lat: loc.Lat,
		Lng: loc.Lng,
	}, h.resolution)

	if err != nil {
		return 0, err
	}

	return cell, nil
}

func (h *H3Index) Insert(driver domain.Driver) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	cell, err := h.cellForLocation(driver.Location)

	if err != nil {
		return err
	}

	if _, ok := h.cellToDrivers[cell]; !ok {
		h.cellToDrivers[cell] = make(map[string]domain.Driver)
	}

	h.cellToDrivers[cell][driver.ID] = driver
	h.driverToCell[driver.ID] = cell
	//O(1) so thi is hot-path safe

	return nil
}

func (h *H3Index) Update(driver domain.Driver) {
	h.mu.Lock()
	defer h.mu.Unlock()

	newCell, err := h.cellForLocation(driver.Location)
	if err != nil {
		return
	}
	oldCell, exists := h.driverToCell[driver.ID]

	if exists && oldCell == newCell {
		//updating snapshot
		if _, ok := h.cellToDrivers[newCell]; !ok {
			//handling bug of busy Driver Eviction
			h.cellToDrivers[newCell] = make(map[string]domain.Driver)
		}
		h.cellToDrivers[newCell][driver.ID] = driver
		return
	}

	if exists {
		delete(h.cellToDrivers[oldCell], driver.ID) //amortised O(1)
	}

	if _, ok := h.cellToDrivers[newCell]; !ok {
		h.cellToDrivers[newCell] = make(map[string]domain.Driver)
	}

	h.cellToDrivers[newCell][driver.ID] = driver
	h.driverToCell[driver.ID] = newCell

}

// func (h *H3Index) removeFromCell(cell h3.Cell, driverID string) {
// 	drivers := h.cellToDrivers[cell]
// 	for i, id := range drivers {
// 		if id == driverID {
// 			drivers[i] = drivers[len(drivers)-1]             //O(1)
// 			h.cellToDrivers[cell] = drivers[:len(drivers)-1] //len((drivers) = len(dirvers)-1)
// 			break
// 		}
// 	}
// }
//worstCase O(l) - l is the len(list of DriverID), BestCase O(1)
//it uses Swap-Delete, but in new version , storing snap-shot of Driver, instead of Slice

func (h *H3Index) Remove(driverID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cell, ok := h.driverToCell[driverID]
	if !ok {
		return
	}

	/*In Go, calling delete(r.drivers, id) only removes that specific "entry" (the key-value pair). It does not destroy the map itself, even if you delete the very last key.*/
	//we do need to handle the empty map[], can't relay on GC
	delete(h.cellToDrivers[cell], driverID) ///amortised O(1)
	if len(h.cellToDrivers[cell]) == 0 {
		delete(h.cellToDrivers, cell) //deleteing cell -> map[] (empty)
	}
	delete(h.driverToCell, driverID) ///amortised O(1)

}

func (h *H3Index) Nearby(cell h3.Cell, k int) []domain.Driver {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := make([]domain.Driver, 0)

	//Expand in Concentric rings
	cells, err := h3.GridDisk(cell, k)

	if err != nil {
		return nil
	}

	for _, c := range cells {
		driversInCell, ok := h.cellToDrivers[c]
		if !ok {
			continue
		}

		for _, driver := range driversInCell {
			if driver.Status == domain.DriverIdle {
				results = append(results, driver)
			}

		}
	}

	return results
	//O(C+D) - C:number of Cells in K-ring, D:total drivers in those cells

}

func (h *H3Index) CellIDForLocation(loc domain.Location) (h3.Cell, error) {
	cell, err := h.cellForLocation(loc)
	if err != nil {
		return 0, err
	}
	return cell, nil
}

const earthRadiusMeters = 6371000 //mean Earth radius

//DistanceMeters computes great-circle distance using "Haversine" formula.
//Pure Fucntion. no side effects. Safe to call anywhere.>

func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*
			math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	//Atan2 -> tan inverse(x,y) returns from -PI to PI
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}
