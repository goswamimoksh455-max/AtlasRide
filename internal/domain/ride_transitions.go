package domain

import "errors"

var ErrInvalidRideTransition = errors.New("invalid ride status transition")

func ValidateRideTransition(from, to RideStatus) error {
	switch from {
	case RideAssigned:
		if to != RideOngoing && to != RideCanceled {
			return ErrInvalidRideTransition
		}

	case RideOngoing:
		if to != RideCompleted && to != RideCanceled {
			return ErrInvalidRideTransition
		}

	case RideCompleted, RideCanceled:
		return ErrInvalidRideTransition
	}

	return nil
}
