//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

func TestProcessedEventRepo_Integration_MarkAndCheck(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewProcessedEventRepo(db)
	ctx := context.Background()

	eventID := "11111111-1111-4111-8111-111111111111"

	done, err := repo.IsProcessed(ctx, eventID, "test-consumer")
	require.NoError(t, err)
	require.False(t, done, "not marked yet")

	require.NoError(t, repo.MarkProcessed(ctx, eventID, "test-consumer"))

	done, err = repo.IsProcessed(ctx, eventID, "test-consumer")
	require.NoError(t, err)
	require.True(t, done)

	// A different consumer's dedup state is independent -- the same event
	// can legitimately be processed by more than one consumer.
	done, err = repo.IsProcessed(ctx, eventID, "other-consumer")
	require.NoError(t, err)
	require.False(t, done)
}

func TestProcessedEventRepo_Integration_MarkProcessedIsIdempotent(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewProcessedEventRepo(db)
	ctx := context.Background()

	eventID := "22222222-2222-4222-8222-222222222222"
	require.NoError(t, repo.MarkProcessed(ctx, eventID, "test-consumer"))
	// Simulates two concurrent deliveries of the same at-least-once
	// message both reaching MarkProcessed -- must not error.
	require.NoError(t, repo.MarkProcessed(ctx, eventID, "test-consumer"))
}
