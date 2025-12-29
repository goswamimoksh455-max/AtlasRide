package repository

import (
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

// DriverRepository abstracts driver storage.
// Hot path implementations must be in-memory.
type DriverRepository interface {
	Upsert(driver domain.Driver) bool
	Get(id string) (domain.Driver, bool)
	Delete(id string)
	Alll() []domain.Driver
	Expired(ttl time.Duration) []string
	ListByCell(cellId string) []domain.Driver
}
