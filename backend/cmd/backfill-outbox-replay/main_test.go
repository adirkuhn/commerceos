package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/outbox"
)

type stubOutboxReplayRepo struct {
	failed     []outbox.Message
	listErr    error
	resetCalls []uint
	resetErr   error
}

func (s *stubOutboxReplayRepo) ListFailedBySubjectPrefix(_ context.Context, _ string) ([]outbox.Message, error) {
	return s.failed, s.listErr
}

func (s *stubOutboxReplayRepo) ResetToPending(_ context.Context, id uint) error {
	s.resetCalls = append(s.resetCalls, id)
	return s.resetErr
}

func TestRun_RequeuesEveryFailedRowReturnedForThePrefix(t *testing.T) {
	repo := &stubOutboxReplayRepo{failed: []outbox.Message{
		{ID: 1, Subject: "shipping.shipment.fulfillment_signal", OccurredAt: time.Now()},
		{ID: 2, Subject: "shipping.shipment.fulfillment_signal", OccurredAt: time.Now()},
	}}

	n, err := run(context.Background(), repo, "shipping.", false)

	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []uint{1, 2}, repo.resetCalls)
}

func TestRun_DryRunListsButNeverCallsResetToPending(t *testing.T) {
	repo := &stubOutboxReplayRepo{failed: []outbox.Message{
		{ID: 1, Subject: "shipping.shipment.fulfillment_signal", OccurredAt: time.Now()},
	}}

	n, err := run(context.Background(), repo, "shipping.", true)

	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Empty(t, repo.resetCalls, "-dry-run must never write")
}

func TestRun_OneRowsResetFailureDoesNotStopTheRest(t *testing.T) {
	repo := &stubOutboxReplayRepo{
		failed: []outbox.Message{
			{ID: 1, Subject: "shipping.shipment.fulfillment_signal"},
			{ID: 2, Subject: "shipping.shipment.fulfillment_signal"},
		},
		resetErr: errors.New("db unavailable"),
	}

	n, err := run(context.Background(), repo, "shipping.", false)

	require.NoError(t, err, "a per-row reset failure is logged, not fatal to the whole run")
	assert.Equal(t, 2, n)
	assert.Equal(t, []uint{1, 2}, repo.resetCalls, "every row is still attempted despite earlier failures")
}

func TestRun_NoFailedRowsIsANoOp(t *testing.T) {
	repo := &stubOutboxReplayRepo{}

	n, err := run(context.Background(), repo, "shipping.", false)

	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, repo.resetCalls)
}

func TestRun_PropagatesListError(t *testing.T) {
	repo := &stubOutboxReplayRepo{listErr: errors.New("db down")}

	_, err := run(context.Background(), repo, "shipping.", false)

	require.Error(t, err)
}
