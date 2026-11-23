//go:build integration

package repositories_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func newAssistantRequest(t *testing.T, tenantID, title string, at time.Time) *assistant.Request {
	t.Helper()
	r, err := assistant.NewRequest(tenantID, "u-1", "customer_admin", title, at)
	require.NoError(t, err)
	return r
}

func createAssistantRequests(t *testing.T, db *gorm.DB, ctx context.Context, rs ...*assistant.Request) {
	t.Helper()
	repo := repositories.NewAssistantRequestRepo(db)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, r := range rs {
			if err := repo.Create(ctx, r); err != nil {
				return err
			}
		}
		return nil
	}))
}

func TestAssistantRequestRepo_CreateGetUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAssistantRequestRepo(db)
	tenantID, ctx := newTestTenant()
	at := time.Now().UTC().Truncate(time.Microsecond)
	r := newAssistantRequest(t, tenantID, "status of CO-12?", at)
	createAssistantRequests(t, db, ctx, r)
	require.NotEmpty(t, r.ID)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		got, err := repo.GetForUpdate(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, *r, *got)

		require.NoError(t, got.Start(at))
		require.NoError(t, got.RetryLater(assistant.WaitModelBusy, "429", at))
		require.NoError(t, repo.Update(ctx, got))

		again, err := repo.Get(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, assistant.StatusQueued, again.Status)
		assert.Equal(t, assistant.WaitModelBusy, again.WaitReason)
		assert.Equal(t, "429", again.LastError)
		assert.Equal(t, 1, again.Attempts)
		assert.True(t, at.Add(30*time.Second).Equal(again.NextAttemptAt))
		return nil
	})
	require.NoError(t, err)
}

func TestAssistantRequestRepo_GetMissing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repositories.NewAssistantRequestRepo(db).Get(ctx, "00000000-0000-0000-0000-000000000001")
		return err
	})

	assert.ErrorIs(t, err, assistant.ErrRequestNotFound)
}

func TestAssistantRequestRepo_ListNewestFirstWithCursorAndStatus(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAssistantRequestRepo(db)
	tenantID, ctx := newTestTenant()
	base := time.Now().UTC().Truncate(time.Microsecond)
	r1 := newAssistantRequest(t, tenantID, "one", base)
	r2 := newAssistantRequest(t, tenantID, "two", base.Add(time.Minute))
	r3 := newAssistantRequest(t, tenantID, "three", base.Add(2*time.Minute))
	require.NoError(t, r2.Cancel(base))
	createAssistantRequests(t, db, ctx, r1, r2, r3)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		page, err := repo.List(ctx, assistant.ListFilter{Limit: 2})
		require.NoError(t, err)
		require.Len(t, page, 2)
		assert.Equal(t, []string{"three", "two"}, []string{page[0].Title, page[1].Title})

		next, err := repo.List(ctx, assistant.ListFilter{Limit: 2, Cursor: page[1].ID})
		require.NoError(t, err)
		require.Len(t, next, 1)
		assert.Equal(t, "one", next[0].Title)

		cancelled, err := repo.List(ctx, assistant.ListFilter{Limit: 10, Status: assistant.StatusCancelled})
		require.NoError(t, err)
		require.Len(t, cancelled, 1)
		assert.Equal(t, "two", cancelled[0].Title)
		return nil
	})
	require.NoError(t, err)
}

func TestAssistantMessageRepo_AppendAndListInOrder(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	r := newAssistantRequest(t, tenantID, "hi", time.Now().UTC())
	createAssistantRequests(t, db, ctx, r)
	repo := repositories.NewAssistantMessageRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		msgs := []assistant.Message{
			{RequestID: r.ID, TenantID: tenantID, Role: assistant.RoleUser, Content: "status CO-12?"},
			{RequestID: r.ID, TenantID: tenantID, Role: assistant.RoleAssistant,
				ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "get_sales_order", Arguments: `{"number":"CO-12"}`, Signature: "c2ln"}}},
			{RequestID: r.ID, TenantID: tenantID, Role: assistant.RoleTool, ToolCallID: "c1", ToolName: "get_sales_order", Content: `{"status":"shipped"}`},
			{RequestID: r.ID, TenantID: tenantID, Role: assistant.RoleAssistant, Question: true, Notice: assistant.NoticeTooManySteps},
		}
		for i := range msgs {
			require.NoError(t, repo.Append(ctx, &msgs[i]))
			assert.Equal(t, i+1, msgs[i].Seq)
			assert.NotEmpty(t, msgs[i].ID)
		}

		got, err := repo.ListByRequest(ctx, r.ID)
		require.NoError(t, err)
		require.Len(t, got, 4)
		assert.Equal(t, "status CO-12?", got[0].Content)
		assert.Equal(t, []assistant.ToolCall{{ID: "c1", Name: "get_sales_order", Arguments: `{"number":"CO-12"}`, Signature: "c2ln"}}, got[1].ToolCalls)
		assert.Equal(t, "c1", got[2].ToolCallID)
		assert.Equal(t, "get_sales_order", got[2].ToolName)
		assert.True(t, got[3].Question)
		assert.Equal(t, assistant.NoticeTooManySteps, got[3].Notice)
		assert.Empty(t, got[0].ToolCalls)
		return nil
	})
	require.NoError(t, err)
}

func TestAssistantRequestRepo_LockDueAcrossTenantsSkipsLockedRows(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAssistantRequestRepo(db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	tenantA, ctxA := newTestTenant()
	tenantB, ctxB := newTestTenant()
	dueA := newAssistantRequest(t, tenantA, "due A", now.Add(-time.Minute))
	dueB := newAssistantRequest(t, tenantB, "due B", now.Add(-time.Second))
	later := newAssistantRequest(t, tenantA, "later", now)
	later.NextAttemptAt = now.Add(time.Hour)
	answered := newAssistantRequest(t, tenantA, "answered", now.Add(-time.Hour))
	require.NoError(t, answered.Start(now))
	require.NoError(t, answered.Answer(now))
	createAssistantRequests(t, db, ctxA, dueA, later, answered)
	createAssistantRequests(t, db, ctxB, dueB)
	admin := tenant.WithSuperAdmin(context.Background())

	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- tenant.WithTenant(admin, db, func(ctx context.Context) error {
			got, err := repo.LockDue(ctx, now, 1)
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].ID != dueA.ID {
				t.Errorf("first worker should lock the oldest due request, got %+v", got)
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	err := tenant.WithTenant(admin, db, func(ctx context.Context) error {
		got, err := repo.LockDue(ctx, now, 10)
		require.NoError(t, err)
		require.Len(t, got, 1, "the row the other worker holds is skipped, not waited on")
		assert.Equal(t, dueB.ID, got[0].ID)
		return nil
	})
	require.NoError(t, err)
	close(release)
	require.NoError(t, <-done)
}

func TestAssistantRequestRepo_RequeueStale(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAssistantRequestRepo(db)
	tenantID, ctx := newTestTenant()
	now := time.Now().UTC().Truncate(time.Microsecond)
	stale := newAssistantRequest(t, tenantID, "stale", now.Add(-time.Hour))
	require.NoError(t, stale.Start(now.Add(-time.Hour)))
	fresh := newAssistantRequest(t, tenantID, "fresh", now)
	require.NoError(t, fresh.Start(now))
	createAssistantRequests(t, db, ctx, stale, fresh)

	err := tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		n, err := repo.RequeueStale(ctx, now.Add(-10*time.Minute), now)
		require.NoError(t, err)
		assert.EqualValues(t, 1, n)

		got, err := repo.Get(ctx, stale.ID)
		require.NoError(t, err)
		assert.Equal(t, assistant.StatusQueued, got.Status)
		assert.True(t, now.Equal(got.NextAttemptAt))
		still, err := repo.Get(ctx, fresh.ID)
		require.NoError(t, err)
		assert.Equal(t, assistant.StatusProcessing, still.Status)
		return nil
	})
	require.NoError(t, err)
}

func TestAssistantRepos_TenantsCannotSeeEachOther(t *testing.T) {
	super, dsn := newIntegrationDBAndDSN(t)
	app := newAppRoleDB(t, dsn)
	tenantA, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	r := newAssistantRequest(t, tenantA, "secret question", time.Now().UTC())
	createAssistantRequests(t, super, ctxA, r)
	require.NoError(t, tenant.WithTenant(ctxA, super, func(ctx context.Context) error {
		return repositories.NewAssistantMessageRepo(super).Append(ctx,
			&assistant.Message{RequestID: r.ID, TenantID: tenantA, Role: assistant.RoleUser, Content: "secret"})
	}))

	err := tenant.WithTenant(ctxB, app, func(ctx context.Context) error {
		list, err := repositories.NewAssistantRequestRepo(app).List(ctx, assistant.ListFilter{Limit: 10})
		require.NoError(t, err)
		assert.Empty(t, list)
		_, err = repositories.NewAssistantRequestRepo(app).Get(ctx, r.ID)
		assert.ErrorIs(t, err, assistant.ErrRequestNotFound)
		msgs, err := repositories.NewAssistantMessageRepo(app).ListByRequest(ctx, r.ID)
		require.NoError(t, err)
		assert.Empty(t, msgs)
		return nil
	})
	require.NoError(t, err)
}

func TestAssistantProposalRepo_CreateLockUpdateList(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	at := time.Now().UTC().Truncate(time.Microsecond)
	r := newAssistantRequest(t, tenantID, "quote Ana", at)
	createAssistantRequests(t, db, ctx, r)
	repo := repositories.NewAssistantProposalRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		p, err := assistant.NewProposal(tenantID, r.ID, assistant.KindCreateQuote,
			json.RawMessage(`{"name":"Ana"}`), json.RawMessage(`{"total_minor_units":1200}`), at)
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, p))
		require.NotEmpty(t, p.ID)

		got, err := repo.GetForUpdate(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, assistant.ProposalPending, got.Status)
		assert.JSONEq(t, `{"name":"Ana"}`, string(got.Payload))
		assert.JSONEq(t, `{"total_minor_units":1200}`, string(got.Preview))
		assert.Nil(t, got.DecidedAt)

		require.NoError(t, got.Claim(at))
		require.NoError(t, got.Complete("u-1", "q-1", "Q-0012", at))
		require.NoError(t, repo.Update(ctx, got))

		msgs := repositories.NewAssistantMessageRepo(db)
		m := assistant.Message{RequestID: r.ID, TenantID: tenantID, Role: assistant.RoleAssistant,
			Notice: assistant.NoticeProposalConfirmed, ProposalID: p.ID, Content: "created Q-0012"}
		require.NoError(t, msgs.Append(ctx, &m))
		listed, err := msgs.ListByRequest(ctx, r.ID)
		require.NoError(t, err)
		assert.Equal(t, p.ID, listed[0].ProposalID)

		all, err := repo.ListByRequest(ctx, r.ID)
		require.NoError(t, err)
		require.Len(t, all, 1)
		assert.Equal(t, assistant.ProposalConfirmed, all[0].Status)
		assert.Equal(t, "q-1", all[0].ResultID)
		assert.Equal(t, "Q-0012", all[0].ResultRef)
		assert.Equal(t, "u-1", all[0].DecidedBy)
		require.NotNil(t, all[0].DecidedAt)

		_, err = repo.GetForUpdate(ctx, "00000000-0000-0000-0000-000000000009")
		assert.ErrorIs(t, err, assistant.ErrProposalNotFound)

		// A request can now wait on the decision.
		require.NoError(t, r.Start(at))
		require.NoError(t, r.Propose(at))
		return repositories.NewAssistantRequestRepo(db).Update(ctx, r)
	})
	require.NoError(t, err)
}
