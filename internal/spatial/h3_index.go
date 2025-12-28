package spatial

import (
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

func (h *H3Index) Insert(driver domain.Driver) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cell, err := h.cellForLocation(driver.Location)

	if err != nil {
		return
	}

	if _, ok := h.cellToDrivers[cell]; !ok {
		h.cellToDrivers[cell] = make(map[string]domain.Driver)
	}

	h.cellToDrivers[cell][driver.ID] = driver
	h.driverToCell[driver.ID] = cell
	//O(1) so thi is hot-path safe
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

	delete(h.cellToDrivers[cell], driverID) ///amortised O(1)
	delete(h.driverToCell, driverID)        ///amortised O(1)

}

func (h *H3Index) Nearby(cellID string, k int) []domain.Driver {
	h.mu.RLock()
	defer h.mu.RUnlock()

	startCell := h3.CellFromString(cellID)
	results := make([]domain.Driver, 0)

	//Expand in Concentric rings
	cells, err := h3.GridDisk(startCell, k)

	if err != nil {
		return nil
	}

	for _, cell := range cells {
		driversInCell, ok := h.cellToDrivers[cell]
		if !ok {
			continue
		}

		for _, driver := range driversInCell {
			results = append(results, driver)
		}
	}

	return results
	//O(C,D) - C:number of Cells in K-ring, D:total drivers in those cells

}
