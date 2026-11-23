package assistant_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

func pendingProposal(t *testing.T) *assistant.Proposal {
	t.Helper()
	p, err := assistant.NewProposal("t-1", "r-1", assistant.KindCreateQuote, json.RawMessage(`{"name":"Ana"}`), json.RawMessage(`{"total":1}`), t0)
	require.NoError(t, err)
	return p
}

func TestNewProposal(t *testing.T) {
	p := pendingProposal(t)

	assert.Equal(t, assistant.ProposalPending, p.Status)
	assert.Equal(t, assistant.KindCreateQuote, p.Kind)
	assert.JSONEq(t, `{"name":"Ana"}`, string(p.Payload))

	_, err := assistant.NewProposal("t-1", "r-1", "drop_tables", nil, nil, t0)
	assert.ErrorIs(t, err, assistant.ErrUnknownProposalKind)
}

func TestProposal_claimCompleteRecordsResult(t *testing.T) {
	p := pendingProposal(t)

	require.NoError(t, p.Claim(t0))
	assert.Equal(t, assistant.ProposalExecuting, p.Status)
	assert.ErrorIs(t, p.Claim(t0), assistant.ErrProposalNotPending, "a second confirm can't run it twice")

	require.NoError(t, p.Complete("u-2", "q-1", "Q-0012", t0))
	assert.Equal(t, assistant.ProposalConfirmed, p.Status)
	assert.Equal(t, "q-1", p.ResultID)
	assert.Equal(t, "Q-0012", p.ResultRef)
	assert.Equal(t, "u-2", p.DecidedBy)
	require.NotNil(t, p.DecidedAt)
}

func TestProposal_releaseAfterFailureKeepsItPendingWithTheError(t *testing.T) {
	p := pendingProposal(t)
	require.NoError(t, p.Claim(t0))

	require.NoError(t, p.Release("insufficient stock"))

	assert.Equal(t, assistant.ProposalPending, p.Status)
	assert.Equal(t, "insufficient stock", p.LastError)
	require.NoError(t, p.Claim(t0), "the user can try again")
	require.NoError(t, p.Complete("u-1", "q-1", "Q-1", t0))
	assert.Empty(t, p.LastError)
}

func TestProposal_reject(t *testing.T) {
	p := pendingProposal(t)

	require.NoError(t, p.Reject("u-1", t0))

	assert.Equal(t, assistant.ProposalRejected, p.Status)
	assert.Equal(t, "u-1", p.DecidedBy)
	assert.ErrorIs(t, p.Reject("u-1", t0), assistant.ErrProposalNotPending)
	assert.ErrorIs(t, p.Claim(t0), assistant.ErrProposalNotPending)
}

func TestProposal_onlyPendingCanBeSuperseded(t *testing.T) {
	p := pendingProposal(t)
	require.NoError(t, p.Supersede(t0))
	assert.Equal(t, assistant.ProposalSuperseded, p.Status)

	executing := pendingProposal(t)
	require.NoError(t, executing.Claim(t0))
	assert.ErrorIs(t, executing.Supersede(t0), assistant.ErrProposalNotPending)
}

func TestProposal_completeAndReleaseOnlyWhileExecuting(t *testing.T) {
	p := pendingProposal(t)
	assert.ErrorIs(t, p.Complete("u", "x", "y", t0), assistant.ErrProposalNotPending)
	assert.ErrorIs(t, p.Release("x"), assistant.ErrProposalNotPending)
}

func awaiting(t *testing.T) *assistant.Request {
	t.Helper()
	r := queued(t)
	require.NoError(t, r.Start(t0))
	require.NoError(t, r.Propose(t0))
	return r
}

func TestRequest_proposeWaitsForConfirmation(t *testing.T) {
	r := awaiting(t)

	assert.Equal(t, assistant.StatusAwaitingConfirmation, r.Status)
	assert.True(t, assistant.StatusAwaitingConfirmation.Valid())
	assert.ErrorIs(t, queued(t).Propose(t0), assistant.ErrInvalidTransition)
}

func TestRequest_decidedWithFollowUpKeepsWaiting(t *testing.T) {
	r := awaiting(t)

	require.NoError(t, r.Decided(true, t0))
	assert.Equal(t, assistant.StatusAwaitingConfirmation, r.Status)

	require.NoError(t, r.Decided(false, t0))
	assert.Equal(t, assistant.StatusAnswered, r.Status)

	assert.ErrorIs(t, r.Decided(false, t0), assistant.ErrInvalidTransition)
}

func TestRequest_replyAndCancelWhileAwaitingConfirmation(t *testing.T) {
	r := awaiting(t)
	require.NoError(t, r.Reply(t0), "asking for changes queues the request again")
	assert.Equal(t, assistant.StatusQueued, r.Status)

	r = awaiting(t)
	require.NoError(t, r.Cancel(t0))
	assert.Equal(t, assistant.StatusCancelled, r.Status)
}
