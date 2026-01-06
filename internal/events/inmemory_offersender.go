package events

import (
	"log/slog"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type InMemoryOfferSender struct {
	Drivers DriverStore
}

func NewInMemoryOfferSender(drivers DriverStore) *InMemoryOfferSender {
	return &InMemoryOfferSender{Drivers: drivers}
}

func (s *InMemoryOfferSender) SendDriverOffer(driverID, riderID string) error {
	slog.Info("[OFFER_SENT]",
		"driver", driverID,
		"rider", riderID,
	)

	return s.Drivers.HandleDriverResponse(
		driverID,
		riderID,
		domain.Accept,
	)
}
