package repository

import "github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"

type RideRepository interface {
	Create(ride *domain.Ride) error
	Update(ride *domain.Ride) error
	Get(id string) (*domain.Ride, error)
	Delete(id string) error
}
