package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The resolution chain itself is covered by the ResolvePickScanCode tests
// (shared scanCodeResolver); these cover the packing session gate.

func newResolvePack(t *testing.T, session *omsDomain.PackingSession, skuRepo *stubSKURepo) *appoms.ResolvePackScanCodeUseCase {
	return appoms.NewResolvePackScanCodeUseCase(newMockGormDB(t), &stubPackingSessionRepo{findByIDResult: session}, &stubSalesOrderLineItemRepo{},
		skuRepo, &stubBarcodeAliasRepo{}, &stubScaleCodeAliasRepo{}, &stubVariantRepoForPicking{}, &stubProductRepoForPicking{})
}

func TestResolvePackScanCode_resolvesByBarcode(t *testing.T) {
	uc := newResolvePack(t, &omsDomain.PackingSession{ID: "session-1", SalesOrderID: "order-1", Status: omsDomain.PackingSessionStatusInProgress},
		&stubSKURepo{byBarcode: map[string]*pmsDomain.SKU{"BAR-1": {ID: "sku-1", VariantID: "var-1"}}})

	result, err := uc.Execute(context.Background(), "session-1", "BAR-1")

	require.NoError(t, err)
	assert.Equal(t, "sku-1", result.SKUID)
	assert.NotEmpty(t, result.ProductName)
}

func TestResolvePackScanCode_rejectsWhenSessionNotActive(t *testing.T) {
	uc := newResolvePack(t, &omsDomain.PackingSession{ID: "session-1", Status: omsDomain.PackingSessionStatusCompleted}, &stubSKURepo{})

	_, err := uc.Execute(context.Background(), "session-1", "BAR-1")

	assert.ErrorIs(t, err, omsDomain.ErrPackingSessionNotActive)
}

func TestResolvePackScanCode_rejectsEmptyCode(t *testing.T) {
	uc := newResolvePack(t, nil, &stubSKURepo{})

	_, err := uc.Execute(context.Background(), "session-1", "  ")

	assert.ErrorIs(t, err, omsDomain.ErrPackScanCodeOrSKURequired)
}
