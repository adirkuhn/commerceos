package scheduler_test

import (
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// stubPromotionActivationLineRepo implements every method of
// promotionsDomain.PromotionSKURepository -- only ListDueForActivation/
// ListDueForReversal/MarkApplied/MarkReverted are reachable from
// RunPromotionActivationUseCase.Run.
type stubPromotionActivationLineRepo struct {
	dueForActivation []promotionsDomain.PromotionSKU
	dueForReversal   []promotionsDomain.PromotionSKU
	markedApplied    []tenantLineCall
	markedReverted   []tenantLineCall
}

type tenantLineCall struct {
	TenantID string
	LineID   string
}

func (s *stubPromotionActivationLineRepo) Create(context.Context, *promotionsDomain.PromotionSKU) error {
	return nil
}
func (s *stubPromotionActivationLineRepo) ListByPromotion(context.Context, string) ([]promotionsDomain.PromotionSKU, error) {
	return nil, nil
}
func (s *stubPromotionActivationLineRepo) DeleteByPromotion(context.Context, string) error {
	return nil
}
func (s *stubPromotionActivationLineRepo) ListDueForActivation(_ context.Context, _ time.Time) ([]promotionsDomain.PromotionSKU, error) {
	return s.dueForActivation, nil
}
func (s *stubPromotionActivationLineRepo) ListDueForReversal(_ context.Context, _ time.Time) ([]promotionsDomain.PromotionSKU, error) {
	return s.dueForReversal, nil
}
func (s *stubPromotionActivationLineRepo) MarkApplied(ctx context.Context, lineID string, _ *int64, _ *time.Time) error {
	s.markedApplied = append(s.markedApplied, tenantLineCall{TenantID: tenant.IDFromContext(ctx), LineID: lineID})
	return nil
}
func (s *stubPromotionActivationLineRepo) MarkReverted(ctx context.Context, lineID string) error {
	s.markedReverted = append(s.markedReverted, tenantLineCall{TenantID: tenant.IDFromContext(ctx), LineID: lineID})
	return nil
}

type stubPromotionActivationSKULookup struct{}

func (stubPromotionActivationSKULookup) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	return &pmsDomain.SKU{ID: id, PriceMinorUnits: 1000}, nil
}

// stubPromotionActivationSalePrice records which tenant each Apply/Revert
// ran under, proving RunPromotionActivationUseCase dispatches per-line work
// under that line's own tenant, not the cross-tenant super_admin context
// used to list due lines.
type stubPromotionActivationSalePrice struct {
	applyCalls  []tenantLineCall
	revertCalls []tenantLineCall
}

func (s *stubPromotionActivationSalePrice) Apply(ctx context.Context, skuID string, _ int64) (*int64, *time.Time, error) {
	s.applyCalls = append(s.applyCalls, tenantLineCall{TenantID: tenant.IDFromContext(ctx), LineID: skuID})
	return nil, nil, nil
}
func (s *stubPromotionActivationSalePrice) Revert(ctx context.Context, skuID string, _ *int64, _ *time.Time) error {
	s.revertCalls = append(s.revertCalls, tenantLineCall{TenantID: tenant.IDFromContext(ctx), LineID: skuID})
	return nil
}

func TestRunPromotionActivation_ActivatesDueLinesAndRevertsEndedOnesUnderTheirOwnTenant(t *testing.T) {
	lineRepo := &stubPromotionActivationLineRepo{
		dueForActivation: []promotionsDomain.PromotionSKU{
			{ID: "activate-1", TenantID: "tenant-a", SKUID: "sku-a", DiscountType: promotionsDomain.PromotionTypePercentage, DiscountValue: 1000},
		},
		dueForReversal: []promotionsDomain.PromotionSKU{
			{ID: "revert-1", TenantID: "tenant-b", SKUID: "sku-b"},
		},
	}
	salePrice := &stubPromotionActivationSalePrice{}
	applyUC := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 2), stubPromotionActivationSKULookup{}, salePrice, lineRepo, &eventrecordtest.Outbox{})
	revertUC := apppromotions.NewRevertPromotionSKUUseCase(newMockGormDBN(t, 1), salePrice, lineRepo, &eventrecordtest.Outbox{})
	uc := appscheduler.NewRunPromotionActivationUseCase(newMockGormDBN(t, 3), lineRepo, applyUC, revertUC, &stubOpenClearanceLines{}, &stubClearanceFinisher{})

	err := uc.Run(context.Background())

	require.NoError(t, err)
	require.Len(t, salePrice.applyCalls, 1)
	require.Equal(t, "tenant-a", salePrice.applyCalls[0].TenantID)
	require.Len(t, lineRepo.markedApplied, 1)
	require.Equal(t, "tenant-a", lineRepo.markedApplied[0].TenantID)
	require.Equal(t, "activate-1", lineRepo.markedApplied[0].LineID)

	require.Len(t, salePrice.revertCalls, 1)
	require.Equal(t, "tenant-b", salePrice.revertCalls[0].TenantID)
	require.Len(t, lineRepo.markedReverted, 1)
	require.Equal(t, "tenant-b", lineRepo.markedReverted[0].TenantID)
	require.Equal(t, "revert-1", lineRepo.markedReverted[0].LineID)
}

func TestRunPromotionActivation_noDueLinesIsNoOp(t *testing.T) {
	lineRepo := &stubPromotionActivationLineRepo{}
	salePrice := &stubPromotionActivationSalePrice{}
	applyUC := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 0), stubPromotionActivationSKULookup{}, salePrice, lineRepo, &eventrecordtest.Outbox{})
	revertUC := apppromotions.NewRevertPromotionSKUUseCase(newMockGormDBN(t, 0), salePrice, lineRepo, &eventrecordtest.Outbox{})
	uc := appscheduler.NewRunPromotionActivationUseCase(newMockGormDBN(t, 3), lineRepo, applyUC, revertUC, &stubOpenClearanceLines{}, &stubClearanceFinisher{})

	err := uc.Run(context.Background())

	require.NoError(t, err)
	require.Empty(t, salePrice.applyCalls)
	require.Empty(t, salePrice.revertCalls)
}

type stubOpenClearanceLines struct {
	open []promotionsDomain.PromotionSKU
}

func (s *stubOpenClearanceLines) ListOpen(context.Context) ([]promotionsDomain.PromotionSKU, error) {
	return s.open, nil
}

type stubClearanceFinisher struct {
	calls []tenantLineCall
	err   error
}

func (s *stubClearanceFinisher) ExecuteForLine(ctx context.Context, line promotionsDomain.PromotionSKU) (bool, error) {
	s.calls = append(s.calls, tenantLineCall{TenantID: tenant.IDFromContext(ctx), LineID: line.ID})
	return s.err == nil, s.err
}

// Expiry dates passing and batch deletes raise no stock event, so the sweep
// is what ends those clearances -- each line checked under its own tenant,
// one line's failure not stopping the rest.
func TestRunPromotionActivation_ChecksOpenClearanceLinesUnderTheirOwnTenant(t *testing.T) {
	lineRepo := &stubPromotionActivationLineRepo{}
	salePrice := &stubPromotionActivationSalePrice{}
	applyUC := apppromotions.NewApplyPromotionSKUUseCase(newMockGormDBN(t, 0), stubPromotionActivationSKULookup{}, salePrice, lineRepo, &eventrecordtest.Outbox{})
	revertUC := apppromotions.NewRevertPromotionSKUUseCase(newMockGormDBN(t, 0), salePrice, lineRepo, &eventrecordtest.Outbox{})
	open := &stubOpenClearanceLines{open: []promotionsDomain.PromotionSKU{
		{ID: "clear-1", TenantID: "tenant-a"},
		{ID: "clear-2", TenantID: "tenant-b"},
	}}
	finisher := &stubClearanceFinisher{err: errors.New("batch lookup failed")}
	uc := appscheduler.NewRunPromotionActivationUseCase(newMockGormDBN(t, 3), lineRepo, applyUC, revertUC, open, finisher)

	err := uc.Run(context.Background())

	require.NoError(t, err)
	require.Equal(t, []tenantLineCall{{TenantID: "tenant-a", LineID: "clear-1"}, {TenantID: "tenant-b", LineID: "clear-2"}}, finisher.calls)
}
