package events

type DriverOfferEvent struct {
	DriverID string
	RiderID  string
	Attempts int
}

/*
Why this matters

retries
metrics
tracing
persistence later
*/
