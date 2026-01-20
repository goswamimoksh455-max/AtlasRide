package ride

import (
	"context"
	"testing"

	"github.com/goswamimoksh455-max/projects/AtlasRide/internal/domain"
	"github.com/stretchr/testify/require"
)

func Test_RideService_FinalizeAssignment(t *testing.T) {
	repo := NewMemoryRepo()
	service := NewRideService(repo)

	ctx := context.Background()

	ride, created, err := service.FinalizeAssignment(ctx, "r1", "d1")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, domain.RideAssigned, ride.Status)
}
