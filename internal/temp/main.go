// package main

// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"os"
// 	"time"

// 	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
// 	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/events"
// 	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/matching"
// 	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/repository"
// 	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/spatial"
// 	"github.com/redis/go-redis/v9"
// )

// func initLogger() (*os.File, error) {

// 	file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
// 	if err != nil {
// 		return nil, err
// 	}

// 	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
// 		Level: slog.LevelDebug,
// 	})

// 	logger := slog.New(handler)
// 	slog.SetDefault(logger)

// 	return file, nil
// }

// func main() {
// 	fmt.Println("AtlasRider API ,HAR HAR MAHADEV..")
// 	logFile, err := initLogger()
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer logFile.Close()
// 	slog.Info("HAR HAR MAHADEV !")
// 	slog.Info("AtlasRider API started", "version", "1.0.0")

// 	//creating Repository (storage)
// 	driverRepo := repository.NewInMemoryDriverRepository()
// 	spatialIndex := spatial.NewH3Index(9) //9 ~65-70m, 10 ~10m, 8 ~170m
// 	//1-2 ring neighbors ~200-500m coverage
// 	// ? k=20 thinking about it..
// 	ctx := context.Background()
// 	redisClient := redis.NewClient(&redis.Options{
// 		Addr:     "localhost:6379", //address of the redis server
// 		Password: "",               //by default no pass
// 		DB:       0,                //default DB
// 	})
// 	err = redisClient.Ping(ctx).Err()
// 	if err != nil {
// 		panic(err) //handles the connection error
// 	}

// 	//creating Service that depend on it
// 	offerStore := matching.NewRedisOfferGroupStore(redisClient)
// 	dispatcher := events.NewInMemoryDispatcher(offerStore)

// 	matchingService := matching.NewService(
// 		driverRepo,
// 		spatialIndex,
// 		dispatcher,
// 		offerStore,
// 	)

// 	//no coordination required
// 	recoveryService := matching.NewRecoveryService(driverRepo)

// 	//start background recovery
// 	go func() {
// 		ticker := time.NewTicker(2 * time.Second) //type is channel
// 		for range ticker.C {
// 			recoveryService.Recover()
// 		}
// 	}()

// 	//starting our server
// 	//seeding drivers
// 	drivers := []domain.Driver{
// 		{
// 			ID:       "driver-1",
// 			Status:   domain.DriverIdle,
// 			Location: domain.Location{Lat: 12.9716, Lng: 77.5946},
// 		},
// 		{
// 			ID:       "driver-2",
// 			Status:   domain.DriverIdle,
// 			Location: domain.Location{Lat: 12.9720, Lng: 77.5950},
// 		},
// 	}
// 	time.Sleep(time.Second)
// 	for _, d := range drivers {
// 		driverRepo.Upsert(d)

// 		spatialIndex.Insert(d)

// 		fmt.Println("Inserted", d.ID)

// 	}
// 	time.Sleep(time.Second)
// 	// simulating rider request
// 	//note : Res 9   and k=5 means ~1.7 km	good for Dense Urban
// 	req := matching.MatchRequest{
// 		RiderID: "rider-1",
// 		Lat:     12.9717,
// 		Lng:     77.5947,
// 		K:       5,
// 		MaxDist: 500,
// 	}
// 	fmt.Println("matching rider ... ")

// 	result, err := matchingService.Match(req)
// 	if err != nil {
// 		fmt.Println("match failed:", err)
// 		return
// 	}
// 	time.Sleep(time.Second)

// 	//intentionaly did to check working of BG recovery
// 	stuck := drivers[1]
// 	stuck.Status = domain.DriverMatching
// 	driverRepo.TransitionStatus(stuck.ID, domain.DriverMatching) //upsert always update updated at time bhai

// 	fmt.Println("match success!")
// 	fmt.Printf("rider %s matcehd with Driver %s (%.2fm)\n",
// 		result.RiderID,
// 		result.DriverID,
// 		result.Distance,
// 	)

// 	time.Sleep(time.Second)
// 	//inspecting Driver state
// 	d, _ := driverRepo.Get(result.DriverID)
// 	fmt.Println("driver final sttae:", d.Status)

// 	time.Sleep(10 * time.Second)

// }
package temp
