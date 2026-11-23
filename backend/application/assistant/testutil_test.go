package assistant_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

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
