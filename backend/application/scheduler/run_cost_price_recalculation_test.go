package scheduler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubActiveSKULister struct {
	skus []pmsDomain.SKU
	err  error
}

func (s *stubActiveSKULister) ListActive(context.Context) ([]pmsDomain.SKU, error) {
	return s.skus, s.err
}

// stubCostStep records the order and tenant of every step the job runs.
type stubCostStep struct {
	calls *[]string
	err   error
}

func (s stubCostStep) Execute(ctx context.Context, skuID string) (*apppms.RecalculateCostPricesResult, error) {
	*s.calls = append(*s.calls, "recalc:"+skuID+"@"+tenant.IDFromContext(ctx))
	return &apppms.RecalculateCostPricesResult{}, s.err
}

func (s stubCostStep) SyncFromCase(ctx context.Context, skuID string) error {
	*s.calls = append(*s.calls, "sync:"+skuID+"@"+tenant.IDFromContext(ctx))
	return s.err
}

// stubHistoryStep records the cost-price-from-history sync.
type stubHistoryStep struct {
	calls *[]string
	err   error
}

func (s stubHistoryStep) Execute(ctx context.Context, skuID string) error {
	*s.calls = append(*s.calls, "history:"+skuID+"@"+tenant.IDFromContext(ctx))
	return s.err
}

// The nightly job first puts each cost price back on the last receipt in
// the supplier cost history (COM-11), then fills in unit costs from their
// cases -- both before the recalculation, so it sees them the same night.
func TestRunCostPriceRecalculation_reconcilesHistoryAndCasesThenRecalculates(t *testing.T) {
	var calls []string
	step := stubCostStep{calls: &calls}
	skus := &stubActiveSKULister{skus: []pmsDomain.SKU{{ID: "box", TenantID: "t1"}, {ID: "other", TenantID: "t2"}}}
	uc := appscheduler.NewRunCostPriceRecalculationUseCase(newMockGormDB(t), skus, step, step, stubHistoryStep{calls: &calls})

	require.NoError(t, uc.Run(context.Background()))

	assert.Equal(t, []string{
		"history:box@t1", "sync:box@t1", "recalc:box@t1",
		"history:other@t2", "sync:other@t2", "recalc:other@t2",
	}, calls)
}

func TestRunCostPriceRecalculation_oneSKUsFailureDoesNotStopTheRest(t *testing.T) {
	var calls []string
	step := stubCostStep{calls: &calls, err: errBoom}
	skus := &stubActiveSKULister{skus: []pmsDomain.SKU{{ID: "a", TenantID: "t1"}, {ID: "b", TenantID: "t1"}}}
	uc := appscheduler.NewRunCostPriceRecalculationUseCase(newMockGormDB(t), skus, step, step, stubHistoryStep{calls: &calls, err: errBoom})

	require.NoError(t, uc.Run(context.Background()))

	assert.Len(t, calls, 6)
}

func TestRunCostPriceRecalculation_listFailureIsReturned(t *testing.T) {
	var calls []string
	step := stubCostStep{calls: &calls}
	uc := appscheduler.NewRunCostPriceRecalculationUseCase(newMockGormDB(t), &stubActiveSKULister{err: errBoom}, step, step, stubHistoryStep{calls: &calls})

	require.ErrorIs(t, uc.Run(context.Background()), errBoom)
	assert.Empty(t, calls)
}
