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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "", RawDescription: "Acme Widget A", Quantity: 5, UnitCostMinorUnits: 300},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceAlias, matched[0].MatchSource)
	require.NotNil(t, matched[0].SKUID)
	assert.Equal(t, "sku-2", *matched[0].SKUID)
}

func TestMatchExtractedLineItems_aliasKeyedByRawSKUWhenPresent(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-1|abc-123": {SKUID: "sku-3"},
	}}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

	matched, err := uc.Execute(context.Background(), "supplier-1", []procurementDomain.ExtractedLineItem{
		{RawSKU: "UNKNOWN", RawDescription: "Mystery Item", Quantity: 1, UnitCostMinorUnits: 0},
	})

	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, appprocurement.MatchSourceUnmatched, matched[0].MatchSource)
	assert.Nil(t, matched[0].SKUID)
}

func TestMatchExtractedLineItems_propagatesSKURepoError(t *testing.T) {
	skuRepo := &erroringSKURepo{err: errors.New("db down")}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
	uc := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, aliasRepo)

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
