package procurement_test

import (
	"context"
	"errors"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMatchExtractedLineItems_emptySupplierIDSkipsAliasLookup is a
// regression test: supplier_product_aliases.supplier_id is a UUID column,
// so querying it with an empty string (a PO uploaded without a supplier
// picked, see CreatePurchaseOrderFromDocumentUseCase) previously failed the
// whole job with "invalid input syntax for type uuid" instead of just
// treating the item as unmatched.
func TestMatchExtractedLineItems_emptySupplierIDSkipsAliasLookup(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "", []procurementDomain.ExtractedLineItem{
		{RawSKU: "UNKNOWN", RawDescription: "Mystery Item", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	assert.False(t, aliasRepo.findCalled, "must not query aliases with an empty (invalid UUID) supplier id")
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource)
}

func TestMatchExtractedLineItems_exactSKUMatch(t *testing.T) {
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceExactSKU, matched[0].MatchSource)
	require.NotNil(t, matched[0].SKUID)
	assert.Equal(t, "sku-1", *matched[0].SKUID)
}

func TestMatchExtractedLineItems_aliasMatchWhenNoExactSKU(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-1|acme widget a": {SKUID: "sku-2"},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "", RawDescription: "Acme Widget A", Quantity: 5, UnitCostMinorUnits: 300},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceAlias, matched[0].MatchSource)
	require.NotNil(t, matched[0].SKUID)
	assert.Equal(t, "sku-2", *matched[0].SKUID)
}

// TestMatchExtractedLineItems_aliasMatchCarriesOverUnitsPerPurchasedItem is
// the "case of a product" edge case: once a human has linked "Rapadurinha
// 36x25gr" to the unit SKU with a 36x multiplier, the next document with
// the same raw text must auto-link with that multiplier too, not just the
// SKU.
func TestMatchExtractedLineItems_aliasMatchCarriesOverUnitsPerPurchasedItem(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-1|rapadurinha 36x25gr": {SKUID: "sku-unit-1", UnitsPerPurchasedItem: 36},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawDescription: "Rapadurinha 36x25gr", Quantity: 1, UnitCostMinorUnits: 3600},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceAlias, matched[0].MatchSource)
	assert.Equal(t, int64(36), matched[0].UnitsPerPurchasedItem)
}

func TestMatchExtractedLineItems_exactSKUAndNameMatchesDefaultUnitsPerPurchasedItemToOne(t *testing.T) {
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500},
	})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.MatchSourceExactSKU, matched[0].MatchSource)
	assert.Equal(t, int64(1), matched[0].UnitsPerPurchasedItem, "no learned multiplier available for a fresh exact-SKU match")
}

func TestMatchExtractedLineItems_aliasKeyedByRawSKUWhenPresent(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-1|abc-123": {SKUID: "sku-3"},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "ABC-123", RawDescription: "Something else entirely", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.MatchSourceAlias, matched[0].MatchSource)
	assert.Equal(t, "sku-3", *matched[0].SKUID)
}

func TestMatchExtractedLineItems_unmatchedWhenNoSKUOrAlias(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "UNKNOWN", RawDescription: "Mystery Item", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource)
	assert.Nil(t, matched[0].SKUID)
}

// TestMatchExtractedLineItems_nameMatchHandlesRealWorldWordOrderMismatch is
// the reported bug: doc text "Chá Limão, Gengibre e Mel BARÃO 10 saq." vs
// catalog "Cha de Limão, Gengibre e Mel - Barão de Cotegipe" -- neither
// contains the other as a substring (different word order, doc has a
// pack-size suffix the catalog doesn't, catalog has a brand suffix the doc
// doesn't), so a containment search finds nothing. Token-overlap scoring
// must still resolve it against an unrelated decoy product.
func TestMatchExtractedLineItems_nameMatchHandlesRealWorldWordOrderMismatch(t *testing.T) {
	skuRepo := &stubSKURepo{byVariant: map[string][]pmsDomain.SKU{"variant-1": {{ID: "sku-tea-1"}}}}
	productRepo := &stubProductRepo{tokenResult: []pmsDomain.Product{
		{ID: "product-1", Name: "Cha de Limão, Gengibre e Mel - Barão de Cotegipe"},
		{ID: "product-2", Name: "Cha de Camomila - Barão de Cotegipe- Validade 26/09/25"},
	}}
	variantRepo := &stubVariantRepo{byProduct: map[string][]pmsDomain.Variant{"product-1": {{ID: "variant-1", ProductID: "product-1"}}}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, productRepo, variantRepo, &stubAliasRepo{})

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "BAR019", RawDescription: "Chá Limão, Gengibre e Mel BARÃO 10 saq.", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceName, matched[0].MatchSource)
	require.NotNil(t, matched[0].SKUID)
	assert.Equal(t, "sku-tea-1", *matched[0].SKUID)
}

func TestMatchExtractedLineItems_nameMatchSkippedWhenNoCandidateClearlyWins(t *testing.T) {
	skuRepo := &stubSKURepo{byVariant: map[string][]pmsDomain.SKU{"variant-1": {{ID: "sku-1"}}, "variant-2": {{ID: "sku-2"}}}}
	// Two products near-identically named -- neither should win a coin-flip.
	productRepo := &stubProductRepo{tokenResult: []pmsDomain.Product{
		{ID: "product-1", Name: "Generic Widget Blue"},
		{ID: "product-2", Name: "Generic Widget Red"},
	}}
	variantRepo := &stubVariantRepo{byProduct: map[string][]pmsDomain.Variant{
		"product-1": {{ID: "variant-1"}}, "product-2": {{ID: "variant-2"}},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, productRepo, variantRepo, &stubAliasRepo{})

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawDescription: "Generic Widget", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource, "no clear winner between similarly-scoring products must not auto-link")
	assert.Nil(t, matched[0].SKUID)
}

func TestMatchExtractedLineItems_nameMatchSkippedWhenBestCandidateScoresTooLow(t *testing.T) {
	productRepo := &stubProductRepo{tokenResult: []pmsDomain.Product{
		{ID: "product-1", Name: "Completely Unrelated Product Line"},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, productRepo, &stubVariantRepo{}, &stubAliasRepo{})

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawDescription: "Blue Widget", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource, "a low-overlap candidate must not auto-link just for being the only one returned")
	assert.Nil(t, matched[0].SKUID)
}

func TestMatchExtractedLineItems_nameMatchSkippedWhenProductHasMultipleSKUs(t *testing.T) {
	skuRepo := &stubSKURepo{byVariant: map[string][]pmsDomain.SKU{
		"variant-small": {{ID: "sku-small"}}, "variant-large": {{ID: "sku-large"}},
	}}
	productRepo := &stubProductRepo{tokenResult: []pmsDomain.Product{{ID: "product-1", Name: "Cotton T-Shirt"}}}
	variantRepo := &stubVariantRepo{byProduct: map[string][]pmsDomain.Variant{
		"product-1": {{ID: "variant-small"}, {ID: "variant-large"}},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, productRepo, variantRepo, &stubAliasRepo{})

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawDescription: "Cotton T-Shirt", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource, "a multi-variant product can't be safely narrowed to one SKU by name alone")
	assert.Nil(t, matched[0].SKUID)
}

func TestMatchExtractedLineItems_nameMatchNotAttemptedWhenAliasAlreadyMatched(t *testing.T) {
	productRepo := &stubProductRepo{tokenResult: []pmsDomain.Product{{ID: "product-1", Name: "Widget A"}}}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{"supplier-1|widget a": {SKUID: "sku-2"}}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, productRepo, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawDescription: "Widget A", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	assert.Equal(t, appprocurement.MatchSourceAlias, matched[0].MatchSource)
	assert.Nil(t, productRepo.gotTokens, "name search must not even run once an earlier step already matched")
}

func TestMatchExtractedLineItems_propagatesProductRepoError(t *testing.T) {
	productRepo := &stubProductRepo{tokenErr: errors.New("db down")}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, productRepo, &stubVariantRepo{}, &stubAliasRepo{})

	_, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawDescription: "Widget A", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.Error(t, err)
}

func TestMatchExtractedLineItems_propagatesSKURepoError(t *testing.T) {
	skuRepo := &erroringSKURepo{err: errors.New("db down")}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	_, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.Error(t, err)
}

func TestMatchExtractedLineItems_multipleItemsMixedOutcomes(t *testing.T) {
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-1|widget b": {SKUID: "sku-2"},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500},
		{RawSKU: "", RawDescription: "Widget B", Quantity: 5, UnitCostMinorUnits: 300},
		{RawSKU: "WID-999", RawDescription: "Widget Z", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	require.Len(t, matched, 3)
	assert.Equal(t, appprocurement.MatchSourceExactSKU, matched[0].MatchSource)
	assert.Equal(t, appprocurement.MatchSourceAlias, matched[1].MatchSource)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[2].MatchSource)
}

func TestMatchExtractedLineItems_chargeRowsSkipMatchingEntirely(t *testing.T) {
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"SHIP": {ID: "sku-should-never-be-used", SKUCode: "SHIP"}}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "SHIP", RawDescription: "Shipping", Quantity: 1, UnitCostMinorUnits: 1500, LineType: procurementDomain.LineItemTypeShipping},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource)
	assert.Nil(t, matched[0].SKUID)
	assert.False(t, aliasRepo.findCalled, "a charge row has nothing to look up an alias for")
}

func TestMatchExtractedLineItems_mixOfProductAndChargeRows(t *testing.T) {
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500, LineType: procurementDomain.LineItemTypeProduct},
		{RawDescription: "Discount", Quantity: 1, UnitCostMinorUnits: -200, LineType: procurementDomain.LineItemTypeDiscount},
		{RawDescription: "Handling Fee", Quantity: 1, UnitCostMinorUnits: 300, LineType: procurementDomain.LineItemTypeFee},
	})

	require.NoError(t, err)
	require.Len(t, matched, 3)
	assert.Equal(t, appprocurement.MatchSourceExactSKU, matched[0].MatchSource)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[1].MatchSource)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[2].MatchSource)
}

func TestNormalizeRawKey_prefersRawSKUAndNormalizesCase(t *testing.T) {
	assert.Equal(t, "abc-123", appprocurement.NormalizeRawKey("  ABC-123  ", "ignored description"))
	assert.Equal(t, "widget a", appprocurement.NormalizeRawKey("", "  Widget A  "))
	assert.Equal(t, "", appprocurement.NormalizeRawKey("", ""))
}

// erroringSKURepo always returns a non-not-found error from FindByCode, to
// exercise the matching pipeline's error-propagation path distinctly from
// the "not found, fall through to alias" path.
type erroringSKURepo struct{ err error }

func (s *erroringSKURepo) Create(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *erroringSKURepo) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, s.err
}
func (s *erroringSKURepo) FindByCode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, s.err
}
func (s *erroringSKURepo) FindByBarcode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, s.err
}
func (s *erroringSKURepo) ListByVariant(_ context.Context, _ string) ([]pmsDomain.SKU, error) {
	return nil, nil
}
func (s *erroringSKURepo) Update(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *erroringSKURepo) Delete(_ context.Context, _ string) error         { return nil }
func (s *erroringSKURepo) ListWithExpiredSale(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *erroringSKURepo) ExistsWithTaxClass(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (s *erroringSKURepo) ListActive(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *erroringSKURepo) UpdateCostPrice(_ context.Context, _ string, _ int64) error {
	return nil
}
