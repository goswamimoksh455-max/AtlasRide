package temp

import (
	"context"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
)

type TTLSweeper struct {
	offerGroups *repository.OfferGroupStore
	interval    time.Duration
}

func NewTTLSweeper(og *repository.OfferGroupStore) *TTLSweeper {
	return &TTLSweeper{
		offerGroups: og,
		interval:    1 * time.Second,
	}
}

func (s *TTLSweeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.offerGroups.CleanExpiredOffers()
		case <-ctx.Done():
			return
		}
	}
}
