package events

type Dispatcher interface {
	EmitDriverOffer(driverID, riderID string) error
}
