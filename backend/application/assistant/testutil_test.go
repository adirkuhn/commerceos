package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

var now = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func clock() time.Time { return now }

// newMockGormDB mocks only the tenant.WithTenant envelope (any number of
// transactions); the stub repos do the rest.
func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 50; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

func adminCtx() context.Context {
	ctx := tenant.WithUserID(tenant.WithTenantID(context.Background(), "t-1"), "u-1")
	return tenant.WithRole(ctx, tenant.RoleCustomerAdmin)
}

type stubRequests struct {
	byID     map[string]*assistant.Request
	order    []string
	gotList  assistant.ListFilter
	stale    int64
	err      error
	updateFn func(r *assistant.Request)
}

func newStubRequests(rs ...*assistant.Request) *stubRequests {
	s := &stubRequests{byID: map[string]*assistant.Request{}}
	for _, r := range rs {
		cp := *r
		s.byID[r.ID] = &cp
		s.order = append(s.order, r.ID)
	}
	return s
}

func (s *stubRequests) Create(_ context.Context, r *assistant.Request) error {
	if s.err != nil {
		return s.err
	}
	r.ID = "req-new"
	cp := *r
	s.byID[r.ID] = &cp
	s.order = append(s.order, r.ID)
	return nil
}

func (s *stubRequests) Get(_ context.Context, id string) (*assistant.Request, error) {
	r, ok := s.byID[id]
	if !ok {
		return nil, assistant.ErrRequestNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *stubRequests) GetForUpdate(ctx context.Context, id string) (*assistant.Request, error) {
	return s.Get(ctx, id)
}

func (s *stubRequests) List(_ context.Context, f assistant.ListFilter) ([]assistant.Request, error) {
	s.gotList = f
	var out []assistant.Request
	for _, id := range s.order {
		out = append(out, *s.byID[id])
	}
	return out, s.err
}

func (s *stubRequests) Update(_ context.Context, r *assistant.Request) error {
	if s.err != nil {
		return s.err
	}
	cp := *r
	s.byID[r.ID] = &cp
	if s.updateFn != nil {
		s.updateFn(&cp)
	}
	return nil
}

func (s *stubRequests) LockDue(_ context.Context, at time.Time, limit int) ([]assistant.Request, error) {
	var out []assistant.Request
	for _, id := range s.order {
		r := s.byID[id]
		if r.Status == assistant.StatusQueued && !r.NextAttemptAt.After(at) && len(out) < limit {
			out = append(out, *r)
		}
	}
	return out, s.err
}

func (s *stubRequests) RequeueStale(_ context.Context, _, _ time.Time) (int64, error) {
	return s.stale, s.err
}

type stubMessages struct {
	byReq map[string][]assistant.Message
	err   error
}

func newStubMessages() *stubMessages {
	return &stubMessages{byReq: map[string][]assistant.Message{}}
}

func (s *stubMessages) Append(_ context.Context, m *assistant.Message) error {
	if s.err != nil {
		return s.err
	}
	m.Seq = len(s.byReq[m.RequestID]) + 1
	s.byReq[m.RequestID] = append(s.byReq[m.RequestID], *m)
	return nil
}

func (s *stubMessages) ListByRequest(_ context.Context, id string) ([]assistant.Message, error) {
	out := append([]assistant.Message(nil), s.byReq[id]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out, s.err
}

// scriptedModel replays one reply (or error) per call and records turns.
type scriptedModel struct {
	replies []assistant.ChatReply
	errs    []error
	turns   []assistant.ChatTurn
	onCall  func(call int)
}

func (m *scriptedModel) Next(_ context.Context, turn assistant.ChatTurn) (assistant.ChatReply, error) {
	call := len(m.turns)
	m.turns = append(m.turns, turn)
	if m.onCall != nil {
		m.onCall(call)
	}
	if call < len(m.errs) && m.errs[call] != nil {
		return assistant.ChatReply{}, m.errs[call]
	}
	if call >= len(m.replies) {
		return assistant.ChatReply{}, errors.New("scriptedModel: no reply scripted")
	}
	return m.replies[call], nil
}

type stubResolver struct {
	model assistant.ChatModel
	err   error
}

func (r stubResolver) Resolve(context.Context) (assistant.ChatModel, error) { return r.model, r.err }

// stubTool echoes its args and records the ctx identity it ran under.
type stubTool struct {
	name    string
	result  string
	err     error
	gotArgs []string
	gotUser string
	gotRole tenant.Role
	gotTen  string
}

func (t *stubTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{Name: t.name, Description: "test tool", Parameters: map[string]any{"type": "object"}}
}

func (t *stubTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	t.gotArgs = append(t.gotArgs, string(args))
	t.gotUser, t.gotRole, t.gotTen = tenant.UserIDFromContext(ctx), tenant.RoleFromContext(ctx), tenant.IDFromContext(ctx)
	return t.result, t.err
}

func queuedRequest(id string) *assistant.Request {
	return &assistant.Request{ID: id, TenantID: "t-1", UserID: "u-1", UserRole: "customer_admin", Title: "hi",
		Status: assistant.StatusQueued, NextAttemptAt: now, CreatedAt: now, UpdatedAt: now}
}

type stubProposals struct {
	byID  map[string]*assistant.Proposal
	order []string
	err   error
}

func newStubProposals(ps ...*assistant.Proposal) *stubProposals {
	s := &stubProposals{byID: map[string]*assistant.Proposal{}}
	for _, p := range ps {
		cp := *p
		s.byID[p.ID] = &cp
		s.order = append(s.order, p.ID)
	}
	return s
}

func (s *stubProposals) Create(_ context.Context, p *assistant.Proposal) error {
	if s.err != nil {
		return s.err
	}
	p.ID = fmt.Sprintf("p-%d", len(s.order)+1)
	cp := *p
	s.byID[p.ID] = &cp
	s.order = append(s.order, p.ID)
	return nil
}

func (s *stubProposals) GetForUpdate(_ context.Context, id string) (*assistant.Proposal, error) {
	p, ok := s.byID[id]
	if !ok {
		return nil, assistant.ErrProposalNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *stubProposals) ListByRequest(_ context.Context, requestID string) ([]assistant.Proposal, error) {
	var out []assistant.Proposal
	for _, id := range s.order {
		if s.byID[id].RequestID == requestID {
			out = append(out, *s.byID[id])
		}
	}
	return out, s.err
}

func (s *stubProposals) Update(_ context.Context, p *assistant.Proposal) error {
	if s.err != nil {
		return s.err
	}
	cp := *p
	s.byID[p.ID] = &cp
	return nil
}

// stubAction is a ProposalAction recording what it was asked.
type stubAction struct {
	kind       assistant.ProposalKind
	tool       string
	prepareErr error
	execErr    error
	result     appassistant.ActionResult
	gotArgs    []string
	gotPayload string
	gotChoices appassistant.ConfirmChoices
	gotUser    string
}

func (a *stubAction) Kind() assistant.ProposalKind { return a.kind }

func (a *stubAction) Tool() assistant.ToolDefinition {
	return assistant.ToolDefinition{Name: a.tool, Description: "propose", Parameters: map[string]any{"type": "object"}}
}

func (a *stubAction) Prepare(_ context.Context, args json.RawMessage) (json.RawMessage, json.RawMessage, error) {
	a.gotArgs = append(a.gotArgs, string(args))
	if a.prepareErr != nil {
		return nil, nil, a.prepareErr
	}
	return json.RawMessage(`{"input":` + string(args) + `}`), json.RawMessage(`{"shown":true}`), nil
}

func (a *stubAction) Execute(ctx context.Context, payload json.RawMessage, choices appassistant.ConfirmChoices) (appassistant.ActionResult, error) {
	a.gotPayload, a.gotChoices, a.gotUser = string(payload), choices, tenant.UserIDFromContext(ctx)
	return a.result, a.execErr
}
