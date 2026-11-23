package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/domain/assistant"
)

func TestCreateRequest_queuesWithTheOpeningMessageAndRecordsEvent(t *testing.T) {
	reqs, msgs, outbox := newStubRequests(), newStubMessages(), &eventrecordtest.Outbox{}

	r, err := appassistant.NewCreateRequestUseCase(newMockGormDB(t), reqs, msgs, outbox, clock).
		Execute(adminCtx(), "Status of order CO-12?")

	require.NoError(t, err)
	assert.Equal(t, "req-new", r.ID)
	assert.Equal(t, assistant.StatusQueued, r.Status)
	assert.Equal(t, "t-1", r.TenantID)
	assert.Equal(t, "u-1", r.UserID)
	assert.Equal(t, "customer_admin", r.UserRole)
	require.Len(t, msgs.byReq["req-new"], 1)
	assert.Equal(t, assistant.RoleUser, msgs.byReq["req-new"][0].Role)
	assert.Equal(t, "Status of order CO-12?", msgs.byReq["req-new"][0].Content)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "assistant.request.created", event.Subject)
	assert.NotContains(t, event.Payload, "text", "what the user wrote never goes into an event")
}

func TestCreateRequest_rejectsEmptyText(t *testing.T) {
	reqs := newStubRequests()

	_, err := appassistant.NewCreateRequestUseCase(newMockGormDB(t), reqs, newStubMessages(), &eventrecordtest.Outbox{}, clock).
		Execute(adminCtx(), "  ")

	assert.ErrorIs(t, err, assistant.ErrTextRequired)
	assert.Empty(t, reqs.byID)
}

func TestCreateRequest_needsTenantAndUser(t *testing.T) {
	_, err := appassistant.NewCreateRequestUseCase(newMockGormDB(t), newStubRequests(), newStubMessages(), &eventrecordtest.Outbox{}, clock).
		Execute(context.Background(), "hi")

	assert.ErrorIs(t, err, appassistant.ErrNoIdentity)
}

func TestListRequests_passesFilterWithDefaultLimit(t *testing.T) {
	reqs := newStubRequests(queuedRequest("r-1"), queuedRequest("r-2"))

	out, err := appassistant.NewListRequestsUseCase(newMockGormDB(t), reqs).
		Execute(adminCtx(), assistant.ListFilter{Status: assistant.StatusQueued, Cursor: "r-0"})

	require.NoError(t, err)
	assert.Len(t, out.Requests, 2)
	assert.Equal(t, assistant.ListFilter{Status: assistant.StatusQueued, Cursor: "r-0", Limit: appassistant.DefaultListLimit + 1}, reqs.gotList,
		"fetches one extra row to know if there's a next page")
	assert.Empty(t, out.NextCursor)
}

func TestListRequests_nextCursorWhenMoreRows(t *testing.T) {
	reqs := newStubRequests(queuedRequest("r-1"), queuedRequest("r-2"), queuedRequest("r-3"))

	out, err := appassistant.NewListRequestsUseCase(newMockGormDB(t), reqs).Execute(adminCtx(), assistant.ListFilter{Limit: 2})

	require.NoError(t, err)
	require.Len(t, out.Requests, 2)
	assert.Equal(t, "r-2", out.NextCursor)
}

func TestListRequests_rejectsUnknownStatus(t *testing.T) {
	_, err := appassistant.NewListRequestsUseCase(newMockGormDB(t), newStubRequests()).
		Execute(adminCtx(), assistant.ListFilter{Status: "done"})

	assert.ErrorIs(t, err, appassistant.ErrInvalidStatusFilter)
}

func TestGetRequest_returnsConversationWithoutToolResults(t *testing.T) {
	reqs, msgs := newStubRequests(queuedRequest("r-1")), newStubMessages()
	for _, m := range []assistant.Message{
		{RequestID: "r-1", Role: assistant.RoleUser, Content: "status CO-12?"},
		{RequestID: "r-1", Role: assistant.RoleAssistant, ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "get_sales_order"}}},
		{RequestID: "r-1", Role: assistant.RoleTool, ToolCallID: "c1", ToolName: "get_sales_order", Content: `{"secret":"raw"}`},
		{RequestID: "r-1", Role: assistant.RoleAssistant, Content: "CO-12 is shipped."},
	} {
		m := m
		require.NoError(t, msgs.Append(context.Background(), &m))
	}

	d, err := appassistant.NewGetRequestUseCase(newMockGormDB(t), reqs, msgs, newStubProposals()).Execute(adminCtx(), "r-1")

	require.NoError(t, err)
	assert.Equal(t, "r-1", d.Request.ID)
	require.Len(t, d.Messages, 4)
	assert.Empty(t, d.Messages[2].Content, "raw tool output stays server-side; the UI shows which tool ran")
	assert.Equal(t, "get_sales_order", d.Messages[2].ToolName)
}

func TestGetRequest_notFound(t *testing.T) {
	_, err := appassistant.NewGetRequestUseCase(newMockGormDB(t), newStubRequests(), newStubMessages(), newStubProposals()).Execute(adminCtx(), "nope")

	assert.ErrorIs(t, err, assistant.ErrRequestNotFound)
}

func TestReplyToRequest_appendsAndRequeues(t *testing.T) {
	r := queuedRequest("r-1")
	r.Status = assistant.StatusNeedsInput
	reqs, msgs, outbox := newStubRequests(r), newStubMessages(), &eventrecordtest.Outbox{}

	out, err := appassistant.NewReplyToRequestUseCase(newMockGormDB(t), reqs, msgs, newStubProposals(), outbox, clock).
		Execute(adminCtx(), "r-1", "Ana Bakery")

	require.NoError(t, err)
	assert.Equal(t, assistant.StatusQueued, out.Status)
	assert.Equal(t, assistant.StatusQueued, reqs.byID["r-1"].Status)
	require.Len(t, msgs.byReq["r-1"], 1)
	assert.Equal(t, "Ana Bakery", msgs.byReq["r-1"][0].Content)
	assert.Equal(t, []string{"assistant.request.replied"}, outbox.Subjects())
}

func TestReplyToRequest_notWhileQueued(t *testing.T) {
	reqs, msgs := newStubRequests(queuedRequest("r-1")), newStubMessages()

	_, err := appassistant.NewReplyToRequestUseCase(newMockGormDB(t), reqs, msgs, newStubProposals(), &eventrecordtest.Outbox{}, clock).
		Execute(adminCtx(), "r-1", "more")

	assert.ErrorIs(t, err, assistant.ErrInvalidTransition)
	assert.Empty(t, msgs.byReq["r-1"])
}

func TestReplyToRequest_validatesText(t *testing.T) {
	r := queuedRequest("r-1")
	r.Status = assistant.StatusAnswered

	_, err := appassistant.NewReplyToRequestUseCase(newMockGormDB(t), newStubRequests(r), newStubMessages(), newStubProposals(), &eventrecordtest.Outbox{}, clock).
		Execute(adminCtx(), "r-1", "")

	assert.ErrorIs(t, err, assistant.ErrTextRequired)
}

func TestCancelRequest(t *testing.T) {
	reqs, outbox := newStubRequests(queuedRequest("r-1")), &eventrecordtest.Outbox{}

	out, err := appassistant.NewCancelRequestUseCase(newMockGormDB(t), reqs, newStubProposals(), outbox, clock).Execute(adminCtx(), "r-1")

	require.NoError(t, err)
	assert.Equal(t, assistant.StatusCancelled, out.Status)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "assistant.request.cancelled", event.Subject)
	assert.Equal(t, map[string]any{"previous_status": "queued"}, event.Data())
}

func TestCancelRequest_finishedIsRejected(t *testing.T) {
	r := queuedRequest("r-1")
	r.Status = assistant.StatusAnswered

	_, err := appassistant.NewCancelRequestUseCase(newMockGormDB(t), newStubRequests(r), newStubProposals(), &eventrecordtest.Outbox{}, clock).Execute(adminCtx(), "r-1")

	assert.ErrorIs(t, err, assistant.ErrNotCancellable)
}

func TestCancelRequest_repoErrorIsWrapped(t *testing.T) {
	reqs := newStubRequests(queuedRequest("r-1"))
	reqs.err = errors.New("db down")

	_, err := appassistant.NewCancelRequestUseCase(newMockGormDB(t), reqs, newStubProposals(), &eventrecordtest.Outbox{}, clock).Execute(adminCtx(), "r-1")

	assert.ErrorContains(t, err, "db down")
}

func pendingProposalFor(t *testing.T, requestID string) *assistant.Proposal {
	t.Helper()
	p, err := assistant.NewProposal("t-1", requestID, assistant.KindCreateQuote, json.RawMessage(`{}`), json.RawMessage(`{"total":1}`), now)
	require.NoError(t, err)
	p.ID = "p-1"
	return p
}

func TestGetRequest_includesProposals(t *testing.T) {
	reqs, proposals := newStubRequests(queuedRequest("r-1")), newStubProposals(pendingProposalFor(t, "r-1"))

	d, err := appassistant.NewGetRequestUseCase(newMockGormDB(t), reqs, newStubMessages(), proposals).Execute(adminCtx(), "r-1")

	require.NoError(t, err)
	require.Len(t, d.Proposals, 1)
	assert.JSONEq(t, `{"total":1}`, string(d.Proposals[0].Preview))
	assert.Nil(t, d.Proposals[0].Payload, "the executable payload stays server-side")
}

func TestReplyToRequest_asksForChangesSupersedingThePendingProposal(t *testing.T) {
	r := queuedRequest("r-1")
	r.Status = assistant.StatusAwaitingConfirmation
	proposals, outbox := newStubProposals(pendingProposalFor(t, "r-1")), &eventrecordtest.Outbox{}

	_, err := appassistant.NewReplyToRequestUseCase(newMockGormDB(t), newStubRequests(r), newStubMessages(), proposals, outbox, clock).
		Execute(adminCtx(), "r-1", "make it 12 boxes")

	require.NoError(t, err)
	assert.Equal(t, assistant.ProposalSuperseded, proposals.byID["p-1"].Status)
	assert.Equal(t, map[string]any{"superseded_proposals": float64(1)}, outbox.Events[0].Data())
}

func TestCancelRequest_supersedesThePendingProposal(t *testing.T) {
	r := queuedRequest("r-1")
	r.Status = assistant.StatusAwaitingConfirmation
	proposals := newStubProposals(pendingProposalFor(t, "r-1"))

	_, err := appassistant.NewCancelRequestUseCase(newMockGormDB(t), newStubRequests(r), proposals, &eventrecordtest.Outbox{}, clock).
		Execute(adminCtx(), "r-1")

	require.NoError(t, err)
	assert.Equal(t, assistant.ProposalSuperseded, proposals.byID["p-1"].Status)
}
