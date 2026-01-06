package events

type Dispatcher interface {
	EnqueueDriverOffer(driverID, riderID string) error
	Start()
	Stop()
}
