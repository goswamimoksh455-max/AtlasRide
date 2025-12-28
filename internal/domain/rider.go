package domain

// Rider is request-scoped and stateless.
type Rider struct {
	ID       string
	Location Location
}
