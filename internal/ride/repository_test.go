package ride

import (
	"context"
	"testing"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepo_CreateIfAbsent_Idempotent(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	r1, created1, err := repo.CreateIfAbsent(ctx, "r1", "d1")
	require.NoError(t, err)
	require.True(t, created1)

	r2, created2, err := repo.CreateIfAbsent(ctx, "r1", "d2")
	require.NoError(t, err)
	require.False(t, created2)

	require.Equal(t, r1.ID, r2.ID)
	require.Equal(t, "d1", r2.DriverID)
}

func TestMemoryRepo_GetActiveByRider(t *testing.T) {
	repo := NewMemoryRepo()
	ctx := context.Background()

	_, _, err := repo.CreateIfAbsent(ctx, "r1", "d1")
	require.NoError(t, err)

	ride, ok, err := repo.GetActiveByRider(ctx, "r1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "d1", ride.DriverID)
}

func TestMemoryRepo_ContextCancelled(t *testing.T) {
	repo := NewMemoryRepo()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := repo.CreateIfAbsent(ctx, "r1", "d1")
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestMemoryRepo_CreateIfAbsent_ContextCancelled(t *testing.T) {
	repo := NewMemoryRepo()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := repo.CreateIfAbsent(ctx, "r1", "d1")
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestRideService_FinalizeAssignment(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewRideService(repo)

	ctx := context.Background()

	ride, created, err := svc.FinalizeAssignment(ctx, "r1", "d1")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, domain.RideAssigned, ride.Status)
}
