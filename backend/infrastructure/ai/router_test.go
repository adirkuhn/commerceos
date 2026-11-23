package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

func TestRoute_firstSuccessWins(t *testing.T) {
	var tried []string
	targets := []routeTarget[string]{{name: "a", target: "A"}, {name: "b", target: "B"}, {name: "c", target: "C"}}

	got, err := route(context.Background(), targets, func(s string) (string, error) {
		tried = append(tried, s)
		if s == "A" {
			return "", errors.New("401 invalid api key")
		}
		return "answer from " + s, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "answer from B", got)
	assert.Equal(t, []string{"A", "B"}, tried, "any failure moves to the next; the rest aren't called")
}

func TestRoute_allFailedJoinsEveryReason(t *testing.T) {
	targets := []routeTarget[string]{{name: "nvidia #1", target: "A"}, {name: "gemini #2", target: "B"}}

	_, err := route(context.Background(), targets, func(s string) (int, error) {
		if s == "A" {
			return 0, errors.New("bad key")
		}
		return 0, assistant.ErrModelBusy
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nvidia #1: bad key")
	assert.Contains(t, err.Error(), "gemini #2")
	assert.ErrorIs(t, err, assistant.ErrModelBusy, "a busy connection in the chain makes the whole attempt retryable")
}

func TestRoute_stopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	targets := []routeTarget[string]{{name: "a", target: "A"}, {name: "b", target: "B"}}

	_, err := route(ctx, targets, func(string) (int, error) {
		calls++
		cancel()
		return 0, context.Canceled
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}
