package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestCreateBarcodeAlias_defaultsUnitsPerScanToOne(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	alias, err := uc.Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "sku-1", Code: "0630509892495"})

	require.NoError(t, err)
	assert.Equal(t, int64(1), alias.UnitsPerScan)
	assert.Equal(t, "0630509892495", aliasRepo.created.Code)
}

func TestCreateBarcodeAlias_persistsAnExplicitCaseMultiplier(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	alias, err := uc.Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "sku-1", Code: "CASE-12", UnitsPerScan: 12})

	require.NoError(t, err)
	assert.Equal(t, int64(12), alias.UnitsPerScan)
}

func TestCreateBarcodeAlias_rejectsBlankCode(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "sku-1", Code: "   "})

	require.Error(t, err)
	assert.True(t, errors.Is(err, pmsDomain.ErrBarcodeAliasCodeRequired))
}

func TestCreateBarcodeAlias_rejectsNegativeUnitsPerScan(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "sku-1", Code: "X", UnitsPerScan: -1})

	require.Error(t, err)
	assert.True(t, errors.Is(err, pmsDomain.ErrInvalidUnitsPerScan))
}

func TestCreateBarcodeAlias_propagatesSKUNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "missing", Code: "X"})

	assert.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

func TestCreateBarcodeAlias_duplicateCodePropagates(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubBarcodeAliasRepo{createErr: pmsDomain.ErrDuplicateBarcodeAliasCode}
	uc := apppms.NewCreateBarcodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateBarcodeAliasInput{SKUID: "sku-1", Code: "TAKEN"})

	assert.True(t, errors.Is(err, pmsDomain.ErrDuplicateBarcodeAliasCode))
}

func TestListBarcodeAliases_returnsRepoResult(t *testing.T) {
	aliasRepo := &stubBarcodeAliasRepo{listBySKUResult: []pmsDomain.BarcodeAlias{{ID: "a1"}, {ID: "a2"}}}
	uc := apppms.NewListBarcodeAliasesUseCase(newMockGormDB(t), aliasRepo)

	aliases, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Len(t, aliases, 2)
}

func TestDeleteBarcodeAlias_delegatesToRepo(t *testing.T) {
	aliasRepo := &stubBarcodeAliasRepo{}
	uc := apppms.NewDeleteBarcodeAliasUseCase(newMockGormDB(t), aliasRepo)

	err := uc.Execute(context.Background(), "alias-1")

	require.NoError(t, err)
	assert.Equal(t, "alias-1", aliasRepo.deletedID)
}
