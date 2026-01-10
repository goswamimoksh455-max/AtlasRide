package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/events"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/matching"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/offer"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/ride"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	//Redis setups
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis not reachable: %v", err)
	}

	//Offer Store
	offerStore := offer.NewRedisOfferGroupStore(rdb)

	//Driver Store ( still in memory)
	driverRepo := repository.NewInMemoryDriverRepository()

	//spatial index
	spatialIndex := spatial.NewH3Index(9) //resolution 9 ~ 600m
	RideService := ride.NewRideService(ride.NewMemoryRepo())
	//Matching service
	serice := matching.NewService(
		driverRepo,
		spatialIndex,
		*offerStore,
		RideService,
	)

	//dispatcher
	sender := events.NewInMemoryOfferSender(serice)

	dispatcher := events.NewAsyncDispatcher(
		10,   //workers
		1000, //queue size
		sender,
	)

	serice.SetEventDispatcher(dispatcher)
	dispatcher.Start()

	// Setup HTTP server
	srv := &http.Server{
		Addr: ":8080",
	}

	// Graceful shutdown setup
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Println("Server starting on port 8080")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
	dispatcher.Stop()
	log.Println("Server stopped")
}
