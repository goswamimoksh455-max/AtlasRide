package events

import (
	"log/slog"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

// ResponseHandler is the interface that matching.Service implements
type ResponseHandler interface {
	HandleDriverResponse(driverID string, riderID string, response domain.DriverResponse) error
}

type InMemoryOfferSender struct {
	handler ResponseHandler // Changed from DriverStore to ResponseHandler
}

// NewInMemoryOfferSender accepts any type that can handle driver responses
// In your test, you pass matching.Service which implements this interface
func NewInMemoryOfferSender(handler ResponseHandler) *InMemoryOfferSender {
	return &InMemoryOfferSender{
		handler: handler,
	}
}

func (s *InMemoryOfferSender) SendDriverOffer(driverID, riderID string) error {
	slog.Info("[OFFER_SENDER]",
		"driver", driverID,
		"rider", riderID,
	)

	// Simulate driver accepting (100% accept for testing)
	// In production, this would send a push notification and wait for real response
	response := domain.Accept

	err := s.handler.HandleDriverResponse(driverID, riderID, response)
	if err != nil {
		slog.Error("[OFFER_SENDER] handler failed",
			"driver", driverID,
			"rider", riderID,
			"error", err,
		)
		return err
	}

	return nil
}
