package ride

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
)

type PostgresRideRepository struct {
	db *sql.DB
}

func NewPostgreRideRepository(db *sql.DB) *PostgresRideRepository {
	return &PostgresRideRepository{db: db}
}

func (r *PostgresRideRepository) CreateIfAbsent(
	ctx context.Context,
	riderID string,
	driverID string,
) (domain.Ride, bool, error) {

	now := time.Now().UTC()
	rideID := uuid.New()

	insert := `
	INSERT INTO rides (id, rider_id, driver_id, status, created_at, updated_at)
	VALUES ($1, $2, $3, 'assigned', $4, $5)
	ON CONFLICT ON CONSTRAINT uniq_active_ride_per_rider
	DO NOTHING
	`

	res, err := r.db.ExecContext(
		ctx,
		insert,
		rideID,
		riderID,
		driverID,
		now,
		now,
	)
	if err != nil {
		return domain.Ride{}, false, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return domain.Ride{}, false, err
	}

	// We won
	if rows == 1 {
		return domain.Ride{
			ID:        rideID.String(),
			RiderID:   riderID,
			DriverID:  driverID,
			Status:    domain.RideAssigned,
			CreatedAt: now,
		}, true, nil
	}

	// Someone else already created it
	existing, ok, err := r.GetActiveByRider(ctx, riderID)
	if err != nil {
		return domain.Ride{}, false, err
	}
	if !ok {
		// Extremely rare: constraint lost + race
		return domain.Ride{}, false, domain.ErrRideAlreadyAssigned
	}

	return existing, false, nil
}

func (r *PostgresRideRepository) GetActiveByRider(
	ctx context.Context,
	riderID string,
) (domain.Ride, bool, error) {

	query := `
	SELECT id, rider_id, driver_id, status, created_at, updated_at
	FROM rides
	WHERE rider_id = $1
	  AND status IN ('assigned', 'ongoing')
	LIMIT 1
	`

	var ride domain.Ride

	err := r.db.QueryRowContext(ctx, query, riderID).Scan(
		&ride.ID,
		&ride.RiderID,
		&ride.DriverID,
		&ride.Status,
		&ride.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return domain.Ride{}, false, nil
	}
	if err != nil {
		return domain.Ride{}, false, err
	}

	return ride, true, nil
}
