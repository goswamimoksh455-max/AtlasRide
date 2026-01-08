package matching_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/events"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/matching"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/offer"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
	"github.com/redis/go-redis/v9"
)

func Benchmark_MatchThroughput(b *testing.B) {

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	driverRepo := repository.NewInMemoryDriverRepository()
	offerGroups := offer.NewRedisOfferGroupStore(rdb)
	for i := 0; i < 1000; i++ {
		driverRepo.Upsert(domain.Driver{
			ID:     fmt.Sprintf("driver-%d", i),
			Status: domain.DriverIdle,
			Location: domain.Location{
				Lat: 12.0,
				Lng: 77.0,
			},
			UpdatedAt: time.Now(),
		})
	}

	spatialIndex := spatial.NewH3Index(9)
	service := matching.NewService(driverRepo, spatialIndex, *offerGroups)

	sender := events.NewInMemoryOfferSender(service)
	dispatcher := events.NewAsyncDispatcher(8, 1000, sender)

	service.SetEventDispatcher(dispatcher)
	dispatcher.Start()
	defer dispatcher.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.Match(matching.MatchRequest{
			RiderID: fmt.Sprintf("rider-%d", i),
			Lat:     12.0,
			Lng:     77.0,
			K:       5,
			MaxDist: 5000,
		})
	}
}
