package assistant_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type processFixture struct {
	reqs      *stubRequests
	msgs      *stubMessages
	proposals *stubProposals
	outbox    *eventrecordtest.Outbox
	model     *scriptedModel
	tool      *stubTool
	action    *stubAction
}

func newProcessFixture(t *testing.T) *processFixture {
	t.Helper()
	r := queuedRequest("r-1")
	require.NoError(t, r.Start(now))
	f := &processFixture{
		reqs: newStubRequests(r), msgs: newStubMessages(), proposals: newStubProposals(), outbox: &eventrecordtest.Outbox{},
		model: &scriptedModel{}, tool: &stubTool{name: "get_sales_order", result: `{"number":"CO-12","status":"shipped"}`},
		action: &stubAction{kind: assistant.KindCreateQuote, tool: "propose_quote"},
	}
	require.NoError(t, f.msgs.Append(context.Background(), &assistant.Message{RequestID: "r-1", Role: assistant.RoleUser, Content: "status CO-12?"}))
	return f
}

func (f *processFixture) useCase(t *testing.T, resolver assistant.ChatModelResolver) *appassistant.ProcessRequestUseCase {
	return appassistant.NewProcessRequestUseCase(newMockGormDB(t), appassistant.ProcessDeps{
		Requests: f.reqs, Messages: f.msgs, Proposals: f.proposals, Models: resolver,
		Tools: []assistant.Tool{f.tool}, Actions: []appassistant.ProposalAction{f.action}, Outbox: f.outbox, Now: clock,
	})
}

func (f *processFixture) run(t *testing.T) {
	t.Helper()
	require.NoError(t, f.useCase(t, stubResolver{model: f.model}).Execute(adminCtx(), *f.reqs.byID["r-1"]))
}

func (f *processFixture) request() *assistant.Request { return f.reqs.byID["r-1"] }

func TestProcess_answersDirectly(t *testing.T) {
	f := newProcessFixture(t)
	f.model.replies = []assistant.ChatReply{{Content: "Hello!"}}

	f.run(t)

	assert.Equal(t, assistant.StatusAnswered, f.request().Status)
	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 2)
	assert.Equal(t, assistant.RoleAssistant, msgs[1].Role)
	assert.Equal(t, "Hello!", msgs[1].Content)
	assert.Equal(t, []string{"assistant.request.answered"}, f.outbox.Subjects())

	turn := f.model.turns[0]
	assert.Contains(t, turn.System, "CommerceOS")
	assert.Contains(t, turn.System, "admin_path", "records are linked only with paths the tools returned")
	assert.Len(t, turn.Messages, 1)
	var names []string
	for _, d := range turn.Tools {
		names = append(names, d.Name)
	}
	assert.ElementsMatch(t, []string{"get_sales_order", "propose_quote", "ask_user"}, names)
}

func TestProcess_runsToolsAsTheRequesterThenAnswers(t *testing.T) {
	f := newProcessFixture(t)
	f.model.replies = []assistant.ChatReply{
		{ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "get_sales_order", Arguments: `{"number":"CO-12"}`}}},
		{Content: "CO-12 has shipped."},
	}

	f.run(t)

	assert.Equal(t, []string{`{"number":"CO-12"}`}, f.tool.gotArgs)
	assert.Equal(t, "u-1", f.tool.gotUser)
	assert.Equal(t, tenant.RoleCustomerAdmin, f.tool.gotRole)
	assert.Equal(t, "t-1", f.tool.gotTen)

	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 4)
	assert.Equal(t, assistant.RoleAssistant, msgs[1].Role)
	assert.Equal(t, "c1", msgs[1].ToolCalls[0].ID)
	assert.Equal(t, assistant.RoleTool, msgs[2].Role)
	assert.Equal(t, "c1", msgs[2].ToolCallID)
	assert.Equal(t, `{"number":"CO-12","status":"shipped"}`, msgs[2].Content)
	assert.Equal(t, "CO-12 has shipped.", msgs[3].Content)
	assert.Len(t, f.model.turns[1].Messages, 3, "the second step sees the tool call and its result")

	event, err := f.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"attempts": float64(1), "tool_calls": float64(1)}, event.Data())
}

func TestProcess_toolErrorAndUnknownToolGoBackToTheModel(t *testing.T) {
	f := newProcessFixture(t)
	f.tool.err = errors.New("sales order not found")
	f.model.replies = []assistant.ChatReply{
		{ToolCalls: []assistant.ToolCall{
			{ID: "c1", Name: "get_sales_order", Arguments: `{}`},
			{ID: "c2", Name: "drop_tables", Arguments: `{}`},
		}},
		{Content: "I couldn't find that order."},
	}

	f.run(t)

	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 5)
	assert.Contains(t, msgs[2].Content, "sales order not found")
	assert.Contains(t, msgs[3].Content, "unknown tool")
	assert.Equal(t, assistant.StatusAnswered, f.request().Status)
}

func TestProcess_longToolResultIsTruncated(t *testing.T) {
	f := newProcessFixture(t)
	f.tool.result = strings.Repeat("x", appassistant.MaxToolResultChars+500)
	f.model.replies = []assistant.ChatReply{
		{ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "get_sales_order", Arguments: `{}`}}},
		{Content: "done"},
	}

	f.run(t)

	got := f.msgs.byReq["r-1"][2].Content
	assert.LessOrEqual(t, len(got), appassistant.MaxToolResultChars+100)
	assert.Contains(t, got, "truncated")
}

func TestProcess_askUserPausesForInput(t *testing.T) {
	f := newProcessFixture(t)
	f.model.replies = []assistant.ChatReply{
		{ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "ask_user", Arguments: `{"question":"Which Ana: Ana Bakery or Ana Souza?"}`}}},
	}

	f.run(t)

	assert.Equal(t, assistant.StatusNeedsInput, f.request().Status)
	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 2)
	assert.True(t, msgs[1].Question)
	assert.Equal(t, "Which Ana: Ana Bakery or Ana Souza?", msgs[1].Content)
	assert.Empty(t, msgs[1].ToolCalls, "the question is stored as plain text so the user's reply follows it")
	assert.Equal(t, []string{"assistant.request.question_asked"}, f.outbox.Subjects())
}

func TestProcess_tooManyStepsAsksUserForDetail(t *testing.T) {
	f := newProcessFixture(t)
	for i := 0; i < appassistant.MaxSteps; i++ {
		f.model.replies = append(f.model.replies, assistant.ChatReply{ToolCalls: []assistant.ToolCall{{ID: fmt.Sprint(i), Name: "get_sales_order", Arguments: `{}`}}})
	}

	f.run(t)

	assert.Equal(t, assistant.StatusNeedsInput, f.request().Status)
	msgs := f.msgs.byReq["r-1"]
	last := msgs[len(msgs)-1]
	assert.Equal(t, assistant.NoticeTooManySteps, last.Notice)
	assert.True(t, last.Question)
	assert.Len(t, f.model.turns, appassistant.MaxSteps)
}

func TestProcess_waitsInTheQueue(t *testing.T) {
	cases := []struct {
		name     string
		resolver assistant.ChatModelResolver
		modelErr error
		want     assistant.WaitReason
	}{
		{"no provider connected", stubResolver{}, nil, assistant.WaitNoProvider},
		{"provider can't run the assistant", stubResolver{err: fmt.Errorf("resolve: %w", assistant.ErrProviderUnsupported)}, nil, assistant.WaitProviderUnsupported},
		{"resolver fails", stubResolver{err: errors.New("decrypt key")}, nil, assistant.WaitModelError},
		{"rate limited", nil, fmt.Errorf("nvidia: %w: 429", assistant.ErrModelBusy), assistant.WaitModelBusy},
		{"bad key", nil, errors.New("401 invalid api key"), assistant.WaitModelError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newProcessFixture(t)
			resolver := tc.resolver
			if resolver == nil {
				f.model.errs = []error{tc.modelErr}
				resolver = stubResolver{model: f.model}
			}

			require.NoError(t, f.useCase(t, resolver).Execute(adminCtx(), *f.request()))

			r := f.request()
			assert.Equal(t, assistant.StatusQueued, r.Status)
			assert.Equal(t, tc.want, r.WaitReason)
			assert.Equal(t, now.Add(30*time.Second), r.NextAttemptAt)
			assert.Empty(t, f.outbox.Subjects())
			assert.Len(t, f.msgs.byReq["r-1"], 1, "nothing is added to the conversation")
		})
	}
}

func TestProcess_resumesFromSavedStepsAfterRateLimit(t *testing.T) {
	f := newProcessFixture(t)
	f.model.replies = []assistant.ChatReply{
		{ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "get_sales_order", Arguments: `{}`}}},
	}
	f.model.errs = []error{nil, fmt.Errorf("%w", assistant.ErrModelBusy)}

	f.run(t)

	assert.Equal(t, assistant.StatusQueued, f.request().Status)
	assert.Len(t, f.msgs.byReq["r-1"], 3, "the tool step is kept, so the retry continues from it")
}

func TestProcess_stopsWhenCancelledMidAttempt(t *testing.T) {
	f := newProcessFixture(t)
	f.model.replies = []assistant.ChatReply{{Content: "too late"}}
	f.model.onCall = func(int) {
		r := f.reqs.byID["r-1"]
		require.NoError(t, r.Cancel(now))
	}

	f.run(t)

	assert.Equal(t, assistant.StatusCancelled, f.request().Status)
	assert.Len(t, f.msgs.byReq["r-1"], 1, "nothing recorded after the user cancelled")
	assert.Empty(t, f.outbox.Subjects())
}

func TestProcess_proposalWaitsForConfirmationWithoutWritingAnything(t *testing.T) {
	f := newProcessFixture(t)
	f.model.replies = []assistant.ChatReply{
		{Content: "Here's the quote.", ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "propose_quote", Arguments: `{"name":"Ana"}`}}},
	}

	f.run(t)

	assert.Equal(t, assistant.StatusAwaitingConfirmation, f.request().Status)
	assert.Equal(t, []string{`{"name":"Ana"}`}, f.action.gotArgs)
	assert.Empty(t, f.action.gotPayload, "nothing is executed before a person confirms")

	require.Len(t, f.proposals.order, 1)
	p := f.proposals.byID[f.proposals.order[0]]
	assert.Equal(t, assistant.ProposalPending, p.Status)
	assert.Equal(t, assistant.KindCreateQuote, p.Kind)
	assert.Equal(t, "r-1", p.RequestID)
	assert.Equal(t, "t-1", p.TenantID)
	assert.JSONEq(t, `{"input":{"name":"Ana"}}`, string(p.Payload))
	assert.JSONEq(t, `{"shown":true}`, string(p.Preview))

	msgs := f.msgs.byReq["r-1"]
	require.Len(t, msgs, 3)
	assert.Equal(t, "Here's the quote.", msgs[1].Content)
	assert.Equal(t, "c1", msgs[1].ToolCalls[0].ID)
	assert.Equal(t, assistant.RoleTool, msgs[2].Role)
	assert.Equal(t, "c1", msgs[2].ToolCallID)
	assert.Equal(t, assistant.NoticeProposed, msgs[2].Notice)
	assert.Equal(t, p.ID, msgs[2].ProposalID)
	assert.Contains(t, msgs[2].Content, "waiting for")

	event, err := f.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "assistant.proposal.created", event.Subject)
	assert.Equal(t, map[string]any{"request_id": "r-1", "kind": "create_quote"}, event.Data())
}

func TestProcess_prepareErrorGoesBackToTheModel(t *testing.T) {
	f := newProcessFixture(t)
	f.action.prepareErr = errors.New("sku s-9 is not a SKU of product p-1")
	f.model.replies = []assistant.ChatReply{
		{ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "propose_quote", Arguments: `{}`}}},
		{Content: "I couldn't find that SKU."},
	}

	f.run(t)

	assert.Equal(t, assistant.StatusAnswered, f.request().Status)
	assert.Empty(t, f.proposals.order)
	assert.Contains(t, f.msgs.byReq["r-1"][2].Content, "sku s-9 is not a SKU")
}

func TestProcess_keepsEveryConnectionsReasonInTheLastError(t *testing.T) {
	f := newProcessFixture(t)
	long := "nvidia: 410 Gone " + strings.Repeat("a", 700) + "\ngemini #2: 400 missing thought_signature " + strings.Repeat("b", 700)
	f.model.errs = []error{errors.New(long)}

	f.run(t)

	assert.Equal(t, long, f.request().LastError, "two providers' reasons fit")
}
