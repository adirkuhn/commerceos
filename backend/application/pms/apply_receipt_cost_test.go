package pms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSKURepoForReceiptCost struct {
	findByIDResult *pmsDomain.SKU
	findByIDErr    error
	updateCalls    []struct {
		skuID               string
		costPriceMinorUnits int64
	}
	updateErr error
}

func (s *stubSKURepoForReceiptCost) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return s.findByIDResult, s.findByIDErr
}

func (s *stubSKURepoForReceiptCost) UpdateCostPriceFromReceipt(_ context.Context, skuID string, costPriceMinorUnits int64) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updateCalls = append(s.updateCalls, struct {
		skuID               string
		costPriceMinorUnits int64
	}{skuID, costPriceMinorUnits})
	return nil
}

type stubBatchListerForReceiptCost struct {
	batches []invDomain.Batch
	err     error
}

func (s *stubBatchListerForReceiptCost) Execute(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.batches, s.err
}

type stubCostLedgerRepo struct {
	created []pmsDomain.SKUCostPriceLedgerEntry
	err     error
}

func (s *stubCostLedgerRepo) Create(_ context.Context, e *pmsDomain.SKUCostPriceLedgerEntry) error {
	if s.err != nil {
		return s.err
	}
	e.ID = "ledger-1"
	s.created = append(s.created, *e)
	return nil
}

func (s *stubCostLedgerRepo) ListBySKU(_ context.Context, _ string, _ int) ([]pmsDomain.SKUCostPriceLedgerEntry, error) {
	return nil, nil
}

func TestApplyReceiptCost_blendsExistingKnownCostAndWritesLedgerEntry(t *testing.T) {
	existingAvg := int64(440)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &existingAvg}}
	// batches already reflect the receipt having committed: 100 existing + 50 just received = 150.
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	ledgerRepo := &stubCostLedgerRepo{}
	occurredAt := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: "purchase_order_line_item", ReferenceID: "line-1", OccurredAt: occurredAt,
	})

	require.NoError(t, err)
	require.Len(t, skuRepo.updateCalls, 1)
	assert.Equal(t, int64(467), skuRepo.updateCalls[0].costPriceMinorUnits)
	require.Len(t, ledgerRepo.created, 1)
	entry := ledgerRepo.created[0]
	assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceReceipt, entry.SourceType)
	assert.Equal(t, int64(50), entry.QuantityDelta)
	assert.Equal(t, int64(150), entry.QuantityAfter)
	require.NotNil(t, entry.CostPriceMinorUnits)
	assert.Equal(t, int64(467), *entry.CostPriceMinorUnits)
	require.NotNil(t, entry.InventoryValueMinorUnits)
	assert.Equal(t, int64(150*467), *entry.InventoryValueMinorUnits)
	require.NotNil(t, entry.ReferenceType)
	assert.Equal(t, "purchase_order_line_item", *entry.ReferenceType)
	require.NotNil(t, entry.ReferenceID)
	assert.Equal(t, "line-1", *entry.ReferenceID)
	assert.Equal(t, occurredAt, entry.OccurredAt)
}

func TestApplyReceiptCost_unknownExistingCostMergesAsReceivedCostAndResolvesSource(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: nil, CostPriceSource: pmsDomain.SKUCostPriceSourceUnknown}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	ledgerRepo := &stubCostLedgerRepo{}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: "batch", ReferenceID: "batch-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, skuRepo.updateCalls, 1)
	// UpdateCostPriceFromReceipt itself flips source unknown -> computed;
	// this use case doesn't need to check that separately, just that the
	// blended value is correct (520, per the unknown-merges-as-received rule).
	assert.Equal(t, int64(520), skuRepo.updateCalls[0].costPriceMinorUnits)
}

func TestApplyReceiptCost_skipsWhenCostPriceSourceIsManual(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	batchLister := &stubBatchListerForReceiptCost{}
	ledgerRepo := &stubCostLedgerRepo{}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Empty(t, skuRepo.updateCalls)
	assert.Empty(t, ledgerRepo.created)
}

func TestApplyReceiptCost_rejectsNonPositiveReceivedQuantity(t *testing.T) {
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), &stubSKURepoForReceiptCost{}, &stubCostLedgerRepo{}, &stubBatchListerForReceiptCost{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{SKUID: "sku-1", ReceivedQuantity: 0, LandedUnitCostMinorUnits: 100})

	require.Error(t, err)
}

func TestApplyReceiptCost_propagatesLedgerCreateError(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 50}}}
	ledgerRepo := &stubCostLedgerRepo{err: errors.New("db down")}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 100, OccurredAt: time.Now()})

	require.Error(t, err)
}
