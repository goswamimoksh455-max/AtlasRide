package temp

import (
	"log/slog"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type DriverStore interface {
	HandleDriverResponse(driverID string, riderID string, response domain.DriverResponse) error
}

type InMemoryDispatcher struct {
	Drivers DriverStore
}

func NewInMemoryDispatcher(drivers DriverStore) *InMemoryDispatcher {
	return &InMemoryDispatcher{
		Drivers: drivers,
	}
}

func (d *InMemoryDispatcher) EmitDriverOffer(driverID, riderID string) {
	slog.Info("[OFFER_EMITTED]",
		"driver", driverID,
		"rider", riderID,
	)
	//get the response from the Driver app
	//then send to the repo that it is ready
	//right now accepting all
	err := d.Drivers.HandleDriverResponse(driverID, riderID, domain.Accept)
	if err != nil {
		slog.Error("Failure in Respond to Intent")

	}

}

//dipatcher is to make the API layer state less and non-blocking
//but final update to the Rider that whatever driverID sent by the MAtch() now he or she accpeted or not via Rider service i think
