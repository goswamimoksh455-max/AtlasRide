package repository

import "github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"

// DriverRepository abstracts driver storage.
// Hot path implementations must be in-memory.
type DriverRepository interface {
	Upsert(driver domain.Driver)
	Get(id string) (domain.Driver, bool)
	Delete(id string)
	ListByCell(cellId string) []domain.Driver
}
