package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSKUStoreForCaseCost is an in-memory SKU store: lookups by id and by
// variant, and a recorded cost write.
type stubSKUStoreForCaseCost struct {
	skus       map[string]*pmsDomain.SKU
	updates    map[string]int64
	updateErr  error
	findErr    error
	listErr    error
	listedVars []string
}

func newStubSKUStoreForCaseCost(skus ...pmsDomain.SKU) *stubSKUStoreForCaseCost {
	s := &stubSKUStoreForCaseCost{skus: map[string]*pmsDomain.SKU{}, updates: map[string]int64{}}
	for i := range skus {
		s.skus[skus[i].ID] = &skus[i]
	}
	return s
}

func (s *stubSKUStoreForCaseCost) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	sku, ok := s.skus[id]
	if !ok {
		return nil, pmsDomain.ErrSKUNotFound
	}
	cp := *sku
	return &cp, nil
}

func (s *stubSKUStoreForCaseCost) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	s.listedVars = append(s.listedVars, variantID)
	var out []pmsDomain.SKU
	for _, sku := range s.skus {
		if sku.VariantID == variantID {
			out = append(out, *sku)
		}
	}
	return out, nil
}

func (s *stubSKUStoreForCaseCost) UpdateCostPriceFromReceipt(_ context.Context, id string, cost int64) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updates[id] = cost
	return nil
}

type stubVariantListerForCaseCost struct {
	variants []pmsDomain.Variant
	err      error
}

func (s *stubVariantListerForCaseCost) ListByProduct(_ context.Context, _ string) ([]pmsDomain.Variant, error) {
	return s.variants, s.err
}

type stubBatchListerBySKU struct {
	batches map[string][]invDomain.Batch
	err     error
}

func (s *stubBatchListerBySKU) Execute(_ context.Context, skuID string) ([]invDomain.Batch, error) {
	return s.batches[skuID], s.err
}

func int64p(v int64) *int64 { return &v }

// box20 is a 20-unit case of "unit", costing 13.99.
func box20(cost *int64) pmsDomain.SKU {
	return pmsDomain.SKU{ID: "box", VariantID: "v-box", CaseOfSKUID: strp("unit"), UnitsPerCase: int64p(20),
		CostPriceMinorUnits: cost, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed}
}

func unitSKU(cost *int64, source pmsDomain.SKUCostPriceSource) pmsDomain.SKU {
	return pmsDomain.SKU{ID: "unit", VariantID: "v-unit", CostPriceMinorUnits: cost, CostPriceSource: source}
}

type caseCostFixture struct {
	skus    *stubSKUStoreForCaseCost
	batches *stubBatchListerBySKU
	ledger  *stubCostLedgerRepo
	outbox  *eventrecordtest.Outbox
	uc      *apppms.SyncUnitCostFromCaseUseCase
}

func newCaseCostFixture(t *testing.T, skus ...pmsDomain.SKU) *caseCostFixture {
	f := &caseCostFixture{
		skus:    newStubSKUStoreForCaseCost(skus...),
		batches: &stubBatchListerBySKU{batches: map[string][]invDomain.Batch{}},
		ledger:  &stubCostLedgerRepo{},
		outbox:  &eventrecordtest.Outbox{},
	}
	variants := &stubVariantListerForCaseCost{variants: []pmsDomain.Variant{{ID: "v-box"}}}
	f.uc = apppms.NewSyncUnitCostFromCaseUseCase(newMockGormDB(t), f.skus, variants, f.ledger, f.batches, f.outbox)
	return f
}

func TestSyncUnitCostFromCase_setsMissingUnitCostFromCase(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1399)), unitSKU(nil, pmsDomain.SKUCostPriceSourceUnknown))

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Equal(t, map[string]int64{"unit": 70}, f.skus.updates)
	require.Len(t, f.ledger.created, 1)
	entry := f.ledger.created[0]
	assert.Equal(t, "unit", entry.SKUID)
	assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceCaseLink, entry.SourceType)
	assert.Equal(t, int64(0), entry.QuantityDelta)
	assert.Equal(t, int64(0), entry.QuantityAfter)
	require.NotNil(t, entry.CostPriceMinorUnits)
	assert.Equal(t, int64(70), *entry.CostPriceMinorUnits)
	require.NotNil(t, entry.ReferenceType)
	assert.Equal(t, "sku", *entry.ReferenceType)
	require.NotNil(t, entry.ReferenceID)
	assert.Equal(t, "box", *entry.ReferenceID)

	ev, err := f.outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "pms.sku.cost_price_changed", ev.Subject)
	assert.Equal(t, "unit", ev.Payload["entity_id"])
	assert.Equal(t, "case_link", ev.Payload["data"].(map[string]any)["reason"])
	assert.Equal(t, [2]any{nil, float64(70)}, ev.Changes()["cost_price_minor_units"])
}

func TestSyncUnitCostFromCase_followsLatestCaseCostWhileNoUnitsOnHand(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1600)), unitSKU(int64p(70), pmsDomain.SKUCostPriceSourceComputed))

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Equal(t, map[string]int64{"unit": 80}, f.skus.updates)
	require.Len(t, f.ledger.created, 1)
}

func TestSyncUnitCostFromCase_keepsUnitCostWhileUnitsAreOnHand(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1600)), unitSKU(int64p(70), pmsDomain.SKUCostPriceSourceComputed))
	f.batches.batches["unit"] = []invDomain.Batch{{Quantity: 5}}

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Empty(t, f.skus.updates)
	assert.Empty(t, f.ledger.created)
	assert.Empty(t, f.outbox.Events)
}

func TestSyncUnitCostFromCase_setsMissingCostEvenWithUnitsOnHand(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1399)), unitSKU(nil, pmsDomain.SKUCostPriceSourceUnknown))
	f.batches.batches["unit"] = []invDomain.Batch{{Quantity: 3}, {Quantity: 2}}

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Equal(t, map[string]int64{"unit": 70}, f.skus.updates)
	require.Len(t, f.ledger.created, 1)
	assert.Equal(t, int64(5), f.ledger.created[0].QuantityAfter)
	require.NotNil(t, f.ledger.created[0].InventoryValueMinorUnits)
	assert.Equal(t, int64(350), *f.ledger.created[0].InventoryValueMinorUnits)
}

func TestSyncUnitCostFromCase_noChangeWhenUnitCostAlreadyMatches(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1399)), unitSKU(int64p(70), pmsDomain.SKUCostPriceSourceComputed))

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Empty(t, f.skus.updates)
	assert.Empty(t, f.ledger.created)
	assert.Empty(t, f.outbox.Events)
}

func TestSyncUnitCostFromCase_neverTouchesManualUnitCost(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1399)), unitSKU(int64p(99), pmsDomain.SKUCostPriceSourceManual))

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Empty(t, f.skus.updates)
	assert.Empty(t, f.outbox.Events)
}

func TestSyncUnitCostFromCase_noOpWhenCaseHasNoCost(t *testing.T) {
	f := newCaseCostFixture(t, box20(nil), unitSKU(nil, pmsDomain.SKUCostPriceSourceUnknown))

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "box"))

	assert.Empty(t, f.skus.updates)
	assert.Empty(t, f.outbox.Events)
}

func TestSyncUnitCostFromCase_noOpWhenSKUIsNotACase(t *testing.T) {
	f := newCaseCostFixture(t, unitSKU(int64p(70), pmsDomain.SKUCostPriceSourceComputed))

	require.NoError(t, f.uc.SyncFromCase(context.Background(), "unit"))

	assert.Empty(t, f.skus.updates)
	assert.Empty(t, f.outbox.Events)
}

func TestSyncUnitCostFromCase_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	tests := map[string]func(f *caseCostFixture){
		"find":   func(f *caseCostFixture) { f.skus.findErr = boom },
		"stock":  func(f *caseCostFixture) { f.batches.err = boom },
		"update": func(f *caseCostFixture) { f.skus.updateErr = boom },
		"ledger": func(f *caseCostFixture) { f.ledger.err = boom },
		"event":  func(f *caseCostFixture) { f.outbox.Err = boom },
	}
	for name, breakIt := range tests {
		t.Run(name, func(t *testing.T) {
			f := newCaseCostFixture(t, box20(int64p(1399)), unitSKU(nil, pmsDomain.SKUCostPriceSourceUnknown))
			breakIt(f)

			err := f.uc.SyncFromCase(context.Background(), "box")

			require.ErrorIs(t, err, boom)
		})
	}
}

func TestSyncUnitCostFromCase_SyncProductSyncsEveryCaseOfTheProduct(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1399)), unitSKU(nil, pmsDomain.SKUCostPriceSourceUnknown))

	require.NoError(t, f.uc.SyncProduct(context.Background(), "p-box"))

	assert.Equal(t, []string{"v-box"}, f.skus.listedVars)
	assert.Equal(t, map[string]int64{"unit": 70}, f.skus.updates)
}

func TestSyncUnitCostFromCase_SyncProductPropagatesListError(t *testing.T) {
	f := newCaseCostFixture(t, box20(int64p(1399)))
	f.skus.listErr = errors.New("boom")

	require.Error(t, f.uc.SyncProduct(context.Background(), "p-box"))
}
