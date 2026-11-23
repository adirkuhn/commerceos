package assistant_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type recordingProcessor struct {
	got  []assistant.Request
	ctxs []context.Context
	err  error
}

func (p *recordingProcessor) Execute(ctx context.Context, r assistant.Request) error {
	p.got = append(p.got, r)
	p.ctxs = append(p.ctxs, ctx)
	return p.err
}

func TestProcessDue_claimsDueRequestsAndRunsEachAsItsRequester(t *testing.T) {
	due := queuedRequest("r-1")
	due.TenantID, due.UserID, due.UserRole = "t-9", "u-9", "customer_owner"
	notYet := queuedRequest("r-2")
	notYet.NextAttemptAt = now.Add(time.Minute)
	reqs, proc := newStubRequests(due, notYet), &recordingProcessor{}

	n, err := appassistant.NewProcessDueRequestsUseCase(newMockGormDB(t), reqs, proc, clock, 5).Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, n)
	require.Len(t, proc.got, 1)
	assert.Equal(t, assistant.StatusProcessing, proc.got[0].Status)
	assert.Equal(t, 1, proc.got[0].Attempts)
	assert.Equal(t, assistant.StatusProcessing, reqs.byID["r-1"].Status, "claimed in the same transaction that locked it")
	assert.Equal(t, assistant.StatusQueued, reqs.byID["r-2"].Status)

	ctx := proc.ctxs[0]
	assert.Equal(t, "t-9", tenant.IDFromContext(ctx))
	assert.False(t, tenant.IsSuperAdmin(ctx), "the attempt runs scoped to the request's tenant, not as super admin")
	assert.Equal(t, "u-9", tenant.UserIDFromContext(ctx))
	assert.Equal(t, tenant.RoleCustomerOwner, tenant.RoleFromContext(ctx))
}

func TestProcessDue_oneFailingRequestDoesNotStopTheRest(t *testing.T) {
	reqs, proc := newStubRequests(queuedRequest("r-1"), queuedRequest("r-2")), &recordingProcessor{err: errors.New("boom")}

	n, err := appassistant.NewProcessDueRequestsUseCase(newMockGormDB(t), reqs, proc, clock, 5).Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Len(t, proc.got, 2)
}

func TestProcessDue_respectsBatchSize(t *testing.T) {
	reqs, proc := newStubRequests(queuedRequest("r-1"), queuedRequest("r-2"), queuedRequest("r-3")), &recordingProcessor{}

	n, err := appassistant.NewProcessDueRequestsUseCase(newMockGormDB(t), reqs, proc, clock, 2).Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestRequeueStaleRequests(t *testing.T) {
	reqs := newStubRequests()
	reqs.stale = 3

	n, err := appassistant.NewRequeueStaleRequestsUseCase(newMockGormDB(t), reqs, clock).Execute(context.Background(), 10*time.Minute)

	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
}
