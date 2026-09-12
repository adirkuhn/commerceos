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

func TestCreateScaleCodeAlias_persistsForTheSecondScale(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubScaleCodeAliasRepo{}
	uc := apppms.NewCreateScaleCodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	alias, err := uc.Execute(context.Background(), apppms.CreateScaleCodeAliasInput{SKUID: "sku-1", Code: "94722"})

	require.NoError(t, err)
	assert.Equal(t, "94722", aliasRepo.created.Code)
	assert.Equal(t, "sku-1", alias.SKUID)
}

func TestCreateScaleCodeAlias_rejectsBlankCode(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubScaleCodeAliasRepo{}
	uc := apppms.NewCreateScaleCodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateScaleCodeAliasInput{SKUID: "sku-1", Code: "  "})

	assert.True(t, errors.Is(err, pmsDomain.ErrScaleCodeAliasCodeRequired))
}

func TestCreateScaleCodeAlias_propagatesSKUNotFound(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
	aliasRepo := &stubScaleCodeAliasRepo{}
	uc := apppms.NewCreateScaleCodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateScaleCodeAliasInput{SKUID: "missing", Code: "94722"})

	assert.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

func TestCreateScaleCodeAlias_duplicateCodePropagates(t *testing.T) {
	skuRepo := &stubSKURepo{}
	aliasRepo := &stubScaleCodeAliasRepo{createErr: pmsDomain.ErrDuplicateScaleCodeAlias}
	uc := apppms.NewCreateScaleCodeAliasUseCase(newMockGormDB(t), skuRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateScaleCodeAliasInput{SKUID: "sku-1", Code: "TAKEN"})

	assert.True(t, errors.Is(err, pmsDomain.ErrDuplicateScaleCodeAlias))
}

func TestListScaleCodeAliases_returnsRepoResult(t *testing.T) {
	aliasRepo := &stubScaleCodeAliasRepo{listBySKUResult: []pmsDomain.ScaleCodeAlias{{ID: "a1"}}}
	uc := apppms.NewListScaleCodeAliasesUseCase(newMockGormDB(t), aliasRepo)

	aliases, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Len(t, aliases, 1)
}

func TestDeleteScaleCodeAlias_delegatesToRepo(t *testing.T) {
	aliasRepo := &stubScaleCodeAliasRepo{}
	uc := apppms.NewDeleteScaleCodeAliasUseCase(newMockGormDB(t), aliasRepo)

	err := uc.Execute(context.Background(), "scalealias-1")

	require.NoError(t, err)
	assert.Equal(t, "scalealias-1", aliasRepo.deletedID)
}
