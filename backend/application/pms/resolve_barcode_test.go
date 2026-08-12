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
	skuRepo := &stubSKURepo{findByBarcodeResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "TSHIRT-1"}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo)

	result, err := uc.Execute(context.Background(), "012345")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, int64(1), result.UnitsPerScan)
}

func TestResolveBarcode_fallsBackToBarcodeAliasWithItsMultiplier(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", SKUCode: "TSHIRT-1"}}
	aliasRepo := &stubBarcodeAliasRepoForResolve{byCode: map[string]*pmsDomain.BarcodeAlias{
		"CASE-BARCODE": {ID: "alias-1", SKUID: "sku-1", Code: "CASE-BARCODE", UnitsPerScan: 12},
	}}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo)

	result, err := uc.Execute(context.Background(), "CASE-BARCODE")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKU.ID)
	assert.Equal(t, int64(12), result.UnitsPerScan)
}

func TestResolveBarcode_returnsSKUNotFoundWhenNeitherMatches(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepoForResolve{}
	uc := apppms.NewResolveBarcodeUseCase(db, skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), "UNKNOWN")

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}
