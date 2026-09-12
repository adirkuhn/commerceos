package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubBarcodeAliasRepoForResolve struct {
	byCode map[string]*pmsDomain.BarcodeAlias
}

func (s *stubBarcodeAliasRepoForResolve) Create(_ context.Context, _ *pmsDomain.BarcodeAlias) error {
	return nil
}
func (s *stubBarcodeAliasRepoForResolve) FindByCode(_ context.Context, code string) (*pmsDomain.BarcodeAlias, error) {
	if a, ok := s.byCode[code]; ok {
		return a, nil
	}
	return nil, pmsDomain.ErrBarcodeAliasNotFound
}
func (s *stubBarcodeAliasRepoForResolve) ListBySKU(_ context.Context, _ string) ([]pmsDomain.BarcodeAlias, error) {
	return nil, nil
}
func (s *stubBarcodeAliasRepoForResolve) Delete(_ context.Context, _ string) error { return nil }

func TestResolveBarcode_resolvesDirectUnitBarcodeWithUnitsPerScanOne(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findByBarcodeResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "TSHIRT-1", VariantID: "variant-1"}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "product-1", Name: "T-Shirt"}}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "012345")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, int64(1), result.UnitsPerScan)
	assert.Equal(t, "T-Shirt", result.ProductName, "scanner apps show the product's name, not just the SKU code")
}

func TestResolveBarcode_fallsBackToBarcodeAliasWithItsMultiplier(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "TSHIRT-1"}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{byCode: map[string]*pmsDomain.BarcodeAlias{
		"CASE-BARCODE": {ID: "alias-1", SKUID: "sku-1", Code: "CASE-BARCODE", UnitsPerScan: 12},
	}}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, &stubVariantRepo{}, &stubProductRepo{product: &pmsDomain.Product{Name: "T-Shirt"}})

	result, err := uc.Execute(context.Background(), "CASE-BARCODE")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, int64(12), result.UnitsPerScan)
	assert.Equal(t, "T-Shirt", result.ProductName)
}

func TestResolveBarcode_fallsBackToSKUCodeWhenNeitherBarcodeNorAliasMatches(t *testing.T) {
	db := newMockGormDB(t)
	// No barcode, no alias -- only a direct SKU.SKUCode match, the "type a
	// SKU code" path the scanner apps' prompt promises.
	skuRepo := &stubSKURepo{findByCodeResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "TSHIRT-1"}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, &stubVariantRepo{}, &stubProductRepo{product: &pmsDomain.Product{Name: "T-Shirt"}})

	result, err := uc.Execute(context.Background(), "TSHIRT-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, int64(1), result.UnitsPerScan)
	assert.Equal(t, "T-Shirt", result.ProductName)
}

func TestResolveBarcode_trimsWhitespaceFromTheScannedCode(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findByBarcodeResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "product-1", Name: "T-Shirt"}}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), " 012345\n")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, "012345", skuRepo.gotBarcode, "a scanner-emitted or manually-typed code can carry stray whitespace -- the exact-match lookup must not silently fail on it")
}

func TestResolveBarcode_resolvesWeightEmbeddedBarcodeFixedPricing(t *testing.T) {
	// Real calabresa label: prefix 29, scale code 00633, weight 333g,
	// valid check digit -- fixed pricing means every package sells for
	// PriceMinorUnits regardless of the scanned weight.
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findByScaleCodeResult: &pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", ScaleCode: strPtr("00633"),
		PricingMode: pmsDomain.SKUPricingModeFixed, PriceMinorUnits: 1999,
	}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}}
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "product-1", Name: "Calabresa"}}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "2900633003332")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, "00633", skuRepo.gotScaleCode)
	require.NotNil(t, result.WeightGrams)
	assert.Equal(t, int64(333), *result.WeightGrams)
	require.NotNil(t, result.LineTotalMinorUnits)
	assert.Equal(t, int64(1999), *result.LineTotalMinorUnits, "fixed pricing ignores the scanned weight")
	assert.Equal(t, int64(1), result.UnitsPerScan)
}

func TestResolveBarcode_resolvesWeightEmbeddedBarcodePerWeightPricing(t *testing.T) {
	// Real dry-meat label: scale code 94721, weight 462g. Total = rate *
	// grams / 1000, rounded to the nearest minor unit:
	// 4999 * 462 / 1000 = 2309.538 -> 2310.
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findByScaleCodeResult: &pmsDomain.SKU{
		ID: "sku-2", VariantID: "variant-2", ScaleCode: strPtr("94721"),
		PricingMode: pmsDomain.SKUPricingModePerWeight, PricePerKiloMinorUnits: int64Ptr(4999),
		PriceMinorUnits: 0,
	}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "variant-2", ProductID: "product-2"}}
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "product-2", Name: "Dry Meat"}}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, variantRepo, productRepo)

	result, err := uc.Execute(context.Background(), "2994721004622")

	require.NoError(t, err)
	require.NotNil(t, result.WeightGrams)
	assert.Equal(t, int64(462), *result.WeightGrams)
	require.NotNil(t, result.LineTotalMinorUnits)
	assert.Equal(t, int64(2310), *result.LineTotalMinorUnits)
}

func TestResolveBarcode_returnsSKUNotFoundWhenScaleCodeUnregistered(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, &stubVariantRepo{}, &stubProductRepo{})

	_, err := uc.Execute(context.Background(), "2900633003332")

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestResolveBarcode_rejectsCorruptedWeightEmbeddedCheckDigit(t *testing.T) {
	// Last digit corrupted (2 -> 9) -- a mis-scanned weight must not
	// silently resolve to a possibly-wrong price, so this must surface as
	// an invalid-barcode error distinct from a plain not-found.
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, &stubVariantRepo{}, &stubProductRepo{})

	_, err := uc.Execute(context.Background(), "2900633003339")

	assert.ErrorIs(t, err, pmsDomain.ErrInvalidCheckDigit)
}

func TestResolveBarcode_returnsSKUNotFoundWhenNothingMatches(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo, &stubVariantRepo{}, &stubProductRepo{})

	_, err := uc.Execute(context.Background(), "UNKNOWN")

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}
