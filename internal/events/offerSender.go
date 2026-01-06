package events

type OfferSender interface {
	SendDriverOffer(driverID, riderID string) error
}
