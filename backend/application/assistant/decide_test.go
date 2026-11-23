package assistant_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/domain/assistant"
)

type decideFixture struct {
	reqs      *stubRequests
	msgs      *stubMessages
	proposals *stubProposals
	outbox    *eventrecordtest.Outbox
	quote     *stubAction
	convert   *stubAction
}

func newDecideFixture(t *testing.T) *decideFixture {
	t.Helper()
	r := queuedRequest("r-1")
	require.NoError(t, r.Start(now))
	require.NoError(t, r.Propose(now))
	p, err := assistant.NewProposal("t-1", "r-1", assistant.KindCreateQuote, json.RawMessage(`{"name":"Ana"}`), json.RawMessage(`{}`), now)
	require.NoError(t, err)
	p.ID = "p-1"
	return &decideFixture{
		reqs: newStubRequests(r), msgs: newStubMessages(), proposals: newStubProposals(p), outbox: &eventrecordtest.Outbox{},
		quote: &stubAction{kind: assistant.KindCreateQuote, tool: "propose_quote", result: appassistant.ActionResult{
			ID: "q-1", Ref: "Q-0012",
			FollowUp: &appassistant.FollowUp{Kind: assistant.KindConvertQuote, Args: json.RawMessage(`{"quote_id":"q-1"}`)},
		}},
		convert: &stubAction{kind: assistant.KindConvertQuote, tool: "propose_quote_conversion", result: appassistant.ActionResult{ID: "so-1", Ref: "CO-40"}},
	}
}

func (f *decideFixture) confirm(t *testing.T) *appassistant.ConfirmProposalUseCase {
	return appassistant.NewConfirmProposalUseCase(newMockGormDB(t), appassistant.DecideDeps{
		Requests: f.reqs, Messages: f.msgs, Proposals: f.proposals,
		Actions: []appassistant.ProposalAction{f.quote, f.convert}, Outbox: f.outbox, Now: clock,
	})
}

func (f *decideFixture) reject(t *testing.T) *appassistant.RejectProposalUseCase {
	return appassistant.NewRejectProposalUseCase(newMockGormDB(t), appassistant.DecideDeps{
		Requests: f.reqs, Messages: f.msgs, Proposals: f.proposals,
		Actions: []appassistant.ProposalAction{f.quote, f.convert}, Outbox: f.outbox, Now: clock,
	})
}

func TestConfirm_runsTheActionAsTheConfirmingUserAndOffersTheFollowUp(t *testing.T) {
	f := newDecideFixture(t)

	d, err := f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{})

	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"Ana"}`, f.quote.gotPayload)
	assert.Equal(t, "u-1", f.quote.gotUser)

	done := f.proposals.byID["p-1"]
	assert.Equal(t, assistant.ProposalConfirmed, done.Status)
	assert.Equal(t, "q-1", done.ResultID)
	assert.Equal(t, "Q-0012", done.ResultRef)
	assert.Equal(t, "u-1", done.DecidedBy)

	require.Len(t, f.proposals.order, 2, "the created quote's conversion is proposed next")
	next := f.proposals.byID[f.proposals.order[1]]
	assert.Equal(t, assistant.KindConvertQuote, next.Kind)
	assert.Equal(t, assistant.ProposalPending, next.Status)
	assert.Equal(t, []string{`{"quote_id":"q-1"}`}, f.convert.gotArgs)
	assert.Equal(t, assistant.StatusAwaitingConfirmation, f.reqs.byID["r-1"].Status)

	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 1)
	assert.Equal(t, assistant.NoticeProposalConfirmed, msgs[0].Notice)
	assert.Equal(t, "p-1", msgs[0].ProposalID)
	assert.Contains(t, msgs[0].Content, "Q-0012", "the model reads what was created on later turns")

	assert.Equal(t, []string{"assistant.proposal.confirmed", "assistant.proposal.created"}, f.outbox.Subjects())
	assert.Equal(t, map[string]any{"request_id": "r-1", "kind": "create_quote", "result_id": "q-1", "result_ref": "Q-0012"}, f.outbox.Events[0].Data())
	assert.Len(t, d.Proposals, 2)
}

func TestConfirm_withoutFollowUpAnswersTheRequest(t *testing.T) {
	f := newDecideFixture(t)
	f.proposals.byID["p-1"].Kind = assistant.KindConvertQuote

	_, err := f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{WarehouseID: "wh-2"})

	require.NoError(t, err)
	assert.Equal(t, appassistant.ConfirmChoices{WarehouseID: "wh-2"}, f.convert.gotChoices)
	assert.Equal(t, assistant.StatusAnswered, f.reqs.byID["r-1"].Status)
	assert.Len(t, f.proposals.order, 1)
}

func TestConfirm_followUpThatCantBePreparedIsSkipped(t *testing.T) {
	f := newDecideFixture(t)
	f.convert.prepareErr = errors.New("no warehouse")

	_, err := f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{})

	require.NoError(t, err, "the quote exists; a follow-up that can't be offered doesn't undo that")
	assert.Equal(t, assistant.StatusAnswered, f.reqs.byID["r-1"].Status)
	assert.Len(t, f.proposals.order, 1)
}

func TestConfirm_failedActionKeepsItPendingWithTheReason(t *testing.T) {
	f := newDecideFixture(t)
	f.quote.execErr = errors.New("a quote needs a name")

	_, err := f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{})

	assert.ErrorContains(t, err, "a quote needs a name")
	p := f.proposals.byID["p-1"]
	assert.Equal(t, assistant.ProposalPending, p.Status, "the user can retry, reject or ask for changes")
	assert.Equal(t, "a quote needs a name", p.LastError)
	assert.Equal(t, assistant.StatusAwaitingConfirmation, f.reqs.byID["r-1"].Status)
	assert.Empty(t, f.outbox.Subjects())
}

func TestConfirm_refusesWhatIsNotPendingOrNotThisRequests(t *testing.T) {
	f := newDecideFixture(t)
	f.proposals.byID["p-1"].Status = assistant.ProposalExecuting

	_, err := f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{})
	assert.ErrorIs(t, err, assistant.ErrProposalNotPending, "a double click doesn't run it twice")
	assert.Empty(t, f.quote.gotPayload)

	f = newDecideFixture(t)
	f.proposals.byID["p-1"].RequestID = "r-other"
	_, err = f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{})
	assert.ErrorIs(t, err, assistant.ErrProposalNotFound)

	f = newDecideFixture(t)
	f.reqs.byID["r-1"].Status = assistant.StatusCancelled
	_, err = f.confirm(t).Execute(adminCtx(), "r-1", "p-1", appassistant.ConfirmChoices{})
	assert.ErrorIs(t, err, assistant.ErrInvalidTransition)
	assert.Empty(t, f.quote.gotPayload)
}

func TestReject(t *testing.T) {
	f := newDecideFixture(t)

	_, err := f.reject(t).Execute(adminCtx(), "r-1", "p-1")

	require.NoError(t, err)
	assert.Equal(t, assistant.ProposalRejected, f.proposals.byID["p-1"].Status)
	assert.Equal(t, assistant.StatusAnswered, f.reqs.byID["r-1"].Status)
	assert.Empty(t, f.quote.gotPayload)
	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 1)
	assert.Equal(t, assistant.NoticeProposalRejected, msgs[0].Notice)
	assert.Equal(t, []string{"assistant.proposal.rejected"}, f.outbox.Subjects())
}

func TestReject_alreadyDecided(t *testing.T) {
	f := newDecideFixture(t)
	f.proposals.byID["p-1"].Status = assistant.ProposalConfirmed

	_, err := f.reject(t).Execute(adminCtx(), "r-1", "p-1")

	assert.ErrorIs(t, err, assistant.ErrProposalNotPending)
}
