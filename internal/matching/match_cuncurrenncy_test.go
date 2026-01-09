package matching_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/events"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/matching"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/offer"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/ride"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
)

func Test_FirstAcceptWins(t *testing.T) {
	ctx := context.Background()

	// ---------- Redis ----------
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Redis connection failed: %v", err)
	}

	_ = rdb.FlushDB(ctx)

	// ---------- Core components ----------
	driverRepo := repository.NewInMemoryDriverRepository()
	offerGroups := offer.NewRedisOfferGroupStore(rdb)
	spatialIndex := spatial.NewH3Index(9)

	RideService := ride.NewRideService(ride.NewMemoryRepo())
	service := matching.NewService(
		driverRepo,
		spatialIndex,
		*offerGroups,
		RideService,
	)

	// ---------- Seed drivers ----------
	const driverCount = 10

	for i := 0; i < driverCount; i++ {
		// Fix driver ID generation
		driver := domain.Driver{
			ID:     fmt.Sprintf("driver-%d", i),
			Status: domain.DriverIdle,
			Location: domain.Location{
				Lat: 12.0,
				Lng: 77.0,
			},
			UpdatedAt: time.Now(),
		}

		ok := driverRepo.Upsert(driver)
		if !ok {
			t.Fatalf("failed to insert driver %s", driver.ID)
		}

		if err := spatialIndex.Insert(driver); err != nil {
			t.Fatalf("spatial insert failed: %v", err)
		}
	}

	// ---------- Dispatcher ----------
	sender := events.NewInMemoryOfferSender(service)
	dispatcher := events.NewAsyncDispatcher(
		4,   // workers
		100, // queue size
		sender,
	)

	service.SetEventDispatcher(dispatcher)
	dispatcher.Start()
	defer dispatcher.Stop()

	// ---------- Execute Match ----------
	result, err := service.Match(matching.MatchRequest{
		RiderID: "rider-1",
		Lat:     12.0,
		Lng:     77.0,
		K:       10,
		MaxDist: 5000,
	})
	if err != nil {
		t.Fatalf("match failed: %v", err)
	}

	t.Logf("Match initiated for rider: %s", result.RiderID)

	// Wait for async processing (increase timeout for reliability)
	time.Sleep(500 * time.Millisecond)

	// ---------- Verify offer group was created ----------
	winner, hasWinner, err := offerGroups.GetWinner(ctx, "rider-1")
	if err != nil {
		t.Fatalf("failed to check winner: %v", err)
	}

	if !hasWinner {
		t.Fatalf("no winner found - offer group may have expired or no driver accepted")
	}

	t.Logf("Winner from Redis: %s", winner)

	// ---------- Assert driver states ----------
	busy := 0
	matching := 0
	idle := 0
	var busyDriver string

	for _, d := range driverRepo.All() {
		switch d.Status {
		case domain.DriverBusy:
			busy++
			busyDriver = d.ID
		case domain.DriverMatching:
			matching++
		case domain.DriverIdle:
			idle++
		}
	}

	t.Logf("Driver states: busy=%d, matching=%d, idle=%d", busy, matching, idle)

	if busy != 1 {
		t.Fatalf("expected exactly 1 busy driver, got %d", busy)
	}

	if busyDriver != winner {
		t.Fatalf("busy driver (%s) doesn't match Redis winner (%s)", busyDriver, winner)
	}

	// Verify losers returned to idle
	if matching > 0 {
		t.Fatalf("expected 0 drivers in matching state, got %d (should have been rejected)", matching)
	}

	t.Logf("PASS: First accept wins → %s", busyDriver)
}

// Test with multiple concurrent riders
func Test_MultipleConcurrentRiders(t *testing.T) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Redis connection failed: %v", err)
	}

	_ = rdb.FlushDB(ctx)

	driverRepo := repository.NewInMemoryDriverRepository()
	offerGroups := offer.NewRedisOfferGroupStore(rdb)
	spatialIndex := spatial.NewH3Index(9)
	RideService := ride.NewRideService(ride.NewMemoryRepo())

	service := matching.NewService(
		driverRepo,
		spatialIndex,
		*offerGroups,
		RideService,
	)

	// Seed 20 drivers
	for i := 0; i < 20; i++ {
		driver := domain.Driver{
			ID:     fmt.Sprintf("driver-%d", i),
			Status: domain.DriverIdle,
			Location: domain.Location{
				Lat: 12.0,
				Lng: 77.0,
			},
			UpdatedAt: time.Now(),
		}
		driverRepo.Upsert(driver)
		spatialIndex.Insert(driver)
	}

	sender := events.NewInMemoryOfferSender(service)
	dispatcher := events.NewAsyncDispatcher(8, 200, sender)
	service.SetEventDispatcher(dispatcher)
	dispatcher.Start()
	defer dispatcher.Stop()

	// Launch 5 concurrent match requests
	const riderCount = 5
	errors := make(chan error, riderCount)

	for i := 0; i < riderCount; i++ {
		go func(idx int) {
			riderID := fmt.Sprintf("rider-%d", idx)
			_, err := service.Match(matching.MatchRequest{
				RiderID: riderID,
				Lat:     12.0,
				Lng:     77.0,
				K:       10,
				MaxDist: 5000,
			})
			errors <- err
		}(i)
	}

	// Collect results
	for i := 0; i < riderCount; i++ {
		if err := <-errors; err != nil && err != matching.ErrNoDriversAvailable {
			t.Fatalf("match %d failed: %v", i, err)
		}
	}

	time.Sleep(1 * time.Second)

	// Verify each rider got unique driver
	winners := make(map[string]string)
	for i := 0; i < riderCount; i++ {
		riderID := fmt.Sprintf("rider-%d", i)
		winner, hasWinner, err := offerGroups.GetWinner(ctx, riderID)
		if err != nil {
			continue
		}
		if hasWinner {
			winners[riderID] = winner
		}
	}

	t.Logf("Matched %d riders successfully", len(winners))

	// Check for duplicate assignments
	seen := make(map[string]bool)
	for rider, driver := range winners {
		if seen[driver] {
			t.Fatalf("driver %s assigned to multiple riders", driver)
		}
		seen[driver] = true
		t.Logf("%s → %s", rider, driver)
	}

	t.Logf(" PASS: No duplicate assignments")
}
