package pms_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeMergeCatalog struct {
	products map[string]pmsDomain.Product
	variants map[string][]pmsDomain.Variant // by product
	skus     map[string][]pmsDomain.SKU     // by variant
	deleted  []string
	costs    map[string]int64 // UpdateCostPriceFromReceipt calls
}

func (f *fakeMergeCatalog) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, pmsDomain.ErrProductNotFound
	}
	return &p, nil
}
func (f *fakeMergeCatalog) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeMergeCatalog) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	return f.variants[productID], nil
}
func (f *fakeMergeCatalog) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return f.skus[variantID], nil
}
func (f *fakeMergeCatalog) UpdateCostPriceFromReceipt(_ context.Context, skuID string, cost int64) error {
	if f.costs == nil {
		f.costs = map[string]int64{}
	}
	f.costs[skuID] = cost
	return nil
}

type fakeOnHand map[string]int64

func (f fakeOnHand) Execute(_ context.Context, skuID string) ([]invDomain.Batch, error) {
	return []invDomain.Batch{{SKUID: skuID, Quantity: f[skuID]}}, nil
}

type fakeSKUReassigner struct {
	calls [][2]string
	err   error
}

func (f *fakeSKUReassigner) ReassignSKU(_ context.Context, from, to string) error {
	f.calls = append(f.calls, [2]string{from, to})
	return f.err
}

type fakeProductReassigner struct{ calls [][2]string }

func (f *fakeProductReassigner) ReassignProduct(_ context.Context, from, to string) error {
	f.calls = append(f.calls, [2]string{from, to})
	return nil
}

type fakeLedger struct {
	entries []pmsDomain.SKUCostPriceLedgerEntry
}

func (f *fakeLedger) Create(_ context.Context, e *pmsDomain.SKUCostPriceLedgerEntry) error {
	f.entries = append(f.entries, *e)
	return nil
}

type fakeCodeAliases struct {
	existing map[string]bool
	barcodes []pmsDomain.BarcodeAlias
	scales   []pmsDomain.ScaleCodeAlias
}

func (f *fakeCodeAliases) CodeTaken(_ context.Context, code string) (bool, error) {
	return f.existing[code], nil
}
func (f *fakeCodeAliases) CreateBarcodeAlias(_ context.Context, a *pmsDomain.BarcodeAlias) error {
	f.barcodes = append(f.barcodes, *a)
	return nil
}
func (f *fakeCodeAliases) CreateScaleCodeAlias(_ context.Context, a *pmsDomain.ScaleCodeAlias) error {
	f.scales = append(f.scales, *a)
	return nil
}

type fakeMergeRecords struct {
	records []pmsDomain.ProductMergeRecord
}

func (f *fakeMergeRecords) Create(_ context.Context, r *pmsDomain.ProductMergeRecord) error {
	f.records = append(f.records, *r)
	return nil
}

type fakeListingLister struct {
	listings []channelsDomain.ProductChannelListing
}

func (f *fakeListingLister) ListByProduct(_ context.Context, _ string) ([]channelsDomain.ProductChannelListing, error) {
	return f.listings, nil
}

type mergeHarness struct {
	catalog   *fakeMergeCatalog
	onHand    fakeOnHand
	inventory *fakeSKUReassigner
	orders    *fakeSKUReassigner
	products  *fakeProductReassigner
	ledger    *fakeLedger
	aliases   *fakeCodeAliases
	records   *fakeMergeRecords
	listings  *fakeListingLister
	outbox    *stubOutboxRepo
	uc        *apppms.MergeProductsUseCase
}

func cost(v int64) *int64 { return &v }

// dup (one SKU d1, barcode 8710001, 4 on hand at 200) merges into orig
// (one SKU o1, 6 on hand at 100).
func newMergeHarness(t *testing.T) *mergeHarness {
	scale := "2100001"
	h := &mergeHarness{
		catalog: &fakeMergeCatalog{
			products: map[string]pmsDomain.Product{
				"dup":  {ID: "dup", Name: "Guarana 350ml (new)"},
				"orig": {ID: "orig", Name: "Guarana Antarctica 350ml"},
			},
			variants: map[string][]pmsDomain.Variant{"dup": {{ID: "vd"}}, "orig": {{ID: "vo"}}},
			skus: map[string][]pmsDomain.SKU{
				"vd": {{ID: "d1", VariantID: "vd", SKUCode: "GUA-NEW", Barcode: "8710001", ScaleCode: &scale, CostPriceMinorUnits: cost(200)}},
				"vo": {{ID: "o1", VariantID: "vo", SKUCode: "GUA-350", Barcode: "7891991", CostPriceMinorUnits: cost(100)}},
			},
		},
		onHand:    fakeOnHand{"d1": 4, "o1": 6},
		inventory: &fakeSKUReassigner{},
		orders:    &fakeSKUReassigner{},
		products:  &fakeProductReassigner{},
		ledger:    &fakeLedger{},
		aliases:   &fakeCodeAliases{existing: map[string]bool{}},
		records:   &fakeMergeRecords{},
		listings:  &fakeListingLister{listings: []channelsDomain.ProductChannelListing{{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "99"}}},
		outbox:    &stubOutboxRepo{},
	}
	h.uc = apppms.NewMergeProductsUseCase(newMockGormDBN(t, 2), apppms.MergeProductsDeps{
		Products: h.catalog, Variants: h.catalog, SKUs: h.catalog, OnHand: h.onHand,
		SKUReassigners:     []apppms.SKUReferenceReassigner{h.inventory, h.orders},
		ProductReassigners: []apppms.ProductReferenceReassigner{h.products},
		Ledger:             h.ledger, CodeAliases: h.aliases, Records: h.records, Listings: h.listings, Outbox: h.outbox,
	})
	return h
}

func mergeCtx() context.Context { return tenant.WithTenantID(context.Background(), "tenant-1") }

// ── tests ────────────────────────────────────────────────────────────────────

func TestMergeProducts_movesEverythingToTheExistingSKUAndDeletesTheDuplicate(t *testing.T) {
	h := newMergeHarness(t)

	result, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig", MergedBy: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, "orig", result.TargetProductID)
	assert.Equal(t, [][2]string{{"d1", "o1"}}, h.inventory.calls)
	assert.Equal(t, [][2]string{{"d1", "o1"}}, h.orders.calls)
	assert.Equal(t, [][2]string{{"dup", "orig"}}, h.products.calls)
	assert.Equal(t, []string{"dup"}, h.catalog.deleted)
	assert.NotContains(t, h.catalog.deleted, "orig", "the existing product is never touched")
}

func TestMergeProducts_blendsCostByOnHandQuantityAndLogsIt(t *testing.T) {
	h := newMergeHarness(t)

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.NoError(t, err)
	// (6*100 + 4*200) / 10 = 140
	assert.Equal(t, map[string]int64{"o1": 140}, h.catalog.costs)
	require.Len(t, h.ledger.entries, 1)
	e := h.ledger.entries[0]
	assert.Equal(t, "o1", e.SKUID)
	assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceMerge, e.SourceType)
	assert.Equal(t, int64(4), e.QuantityDelta)
	assert.Equal(t, int64(10), e.QuantityAfter)
	assert.Equal(t, cost(140), e.CostPriceMinorUnits)
	assert.Equal(t, cost(1400), e.InventoryValueMinorUnits)
}

func TestMergeProducts_leavesAManualCostAlone(t *testing.T) {
	h := newMergeHarness(t)
	h.catalog.skus["vo"][0].CostPriceSource = pmsDomain.SKUCostPriceSourceManual

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.NoError(t, err)
	assert.Empty(t, h.catalog.costs, "a manually set cost is the user's decision, not recomputed")
	require.Len(t, h.ledger.entries, 1)
	assert.Equal(t, cost(100), h.ledger.entries[0].CostPriceMinorUnits)
}

func TestMergeProducts_keepsTheDuplicatesCodesAsAliases(t *testing.T) {
	h := newMergeHarness(t)

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.NoError(t, err)
	require.Len(t, h.aliases.barcodes, 1)
	assert.Equal(t, pmsDomain.BarcodeAlias{TenantID: "tenant-1", SKUID: "o1", Code: "8710001", UnitsPerScan: 1,
		CreatedAt: h.aliases.barcodes[0].CreatedAt, UpdatedAt: h.aliases.barcodes[0].UpdatedAt}, h.aliases.barcodes[0])
	require.Len(t, h.aliases.scales, 1)
	assert.Equal(t, "2100001", h.aliases.scales[0].Code)
	assert.Equal(t, "o1", h.aliases.scales[0].SKUID)
}

func TestMergeProducts_skipsACodeTheExistingSKUAlreadyAnswersTo(t *testing.T) {
	h := newMergeHarness(t)
	h.catalog.skus["vd"][0].Barcode = "7891991" // same barcode as the existing SKU
	h.aliases.existing["2100001"] = true        // scale code already an alias

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.NoError(t, err)
	assert.Empty(t, h.aliases.barcodes)
	assert.Empty(t, h.aliases.scales)
}

func TestMergeProducts_recordsTheMergeAndRaisesEvents(t *testing.T) {
	h := newMergeHarness(t)

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig", MergedBy: "user-1"})

	require.NoError(t, err)
	require.Len(t, h.records.records, 1)
	r := h.records.records[0]
	assert.Equal(t, "Guarana 350ml (new)", r.SourceProductName)
	assert.Equal(t, "user-1", r.MergedBy)
	assert.Equal(t, []pmsDomain.MergedSKU{{SourceSKUID: "d1", SourceSKUCode: "GUA-NEW", SourceBarcode: "8710001", TargetSKUID: "o1"}}, r.SKUs)

	assert.ElementsMatch(t, []string{"pms.product.merged", "pms.product.deleted", "pms.product.changed", "inventory.stock.changed"}, h.outbox.calls)
	for i, subject := range h.outbox.calls {
		if subject == "pms.product.deleted" {
			var ev events.ProductDeleted
			require.NoError(t, json.Unmarshal(h.outbox.payloads[i], &ev))
			assert.Equal(t, "dup", ev.ProductID)
			assert.Len(t, ev.Listings, 1, "the duplicate is removed from the channels it was synced to")
		}
		if subject == "inventory.stock.changed" {
			var ev events.InventoryStockChanged
			require.NoError(t, json.Unmarshal(h.outbox.payloads[i], &ev))
			assert.Equal(t, "o1", ev.SKUID)
		}
	}
}

func TestMergeProducts_explicitMappingForProductsWithSeveralSKUs(t *testing.T) {
	h := newMergeHarness(t)
	h.catalog.skus["vd"] = append(h.catalog.skus["vd"], pmsDomain.SKU{ID: "d2", VariantID: "vd", SKUCode: "GUA-NEW-6"})
	h.catalog.skus["vo"] = append(h.catalog.skus["vo"], pmsDomain.SKU{ID: "o2", VariantID: "vo", SKUCode: "GUA-6"})

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig"})
	require.ErrorIs(t, err, pmsDomain.ErrMergeMappingIncomplete, "no obvious mapping: the user must match SKUs")

	h.inventory.calls = nil
	_, err = h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig", Pairs: []pmsDomain.SKUMergePair{
		{SourceSKUID: "d1", TargetSKUID: "o1"}, {SourceSKUID: "d2", TargetSKUID: "o2"},
	}})
	require.NoError(t, err)
	assert.Equal(t, [][2]string{{"d1", "o1"}, {"d2", "o2"}}, h.inventory.calls)
}

func TestMergeProducts_rejectsInvalidPlansBeforeTouchingAnything(t *testing.T) {
	h := newMergeHarness(t)

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "dup"})
	require.ErrorIs(t, err, pmsDomain.ErrMergeIntoSelf)

	_, err = h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "missing"})
	require.ErrorIs(t, err, pmsDomain.ErrProductNotFound)

	assert.Empty(t, h.inventory.calls)
	assert.Empty(t, h.catalog.deleted)
}

func TestMergeProducts_aReassignFailureAbortsTheMerge(t *testing.T) {
	h := newMergeHarness(t)
	boom := errors.New("boom")
	h.orders.err = boom

	_, err := h.uc.Execute(mergeCtx(), apppms.MergeProductsInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.ErrorIs(t, err, boom)
	assert.Empty(t, h.catalog.deleted, "nothing is deleted once any step fails (the transaction rolls back)")
}
