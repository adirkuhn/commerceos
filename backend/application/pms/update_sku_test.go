package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUpdateSKU_rejectsDimensionsOnVirtualSKU(t *testing.T) {
	skuRepo := &stubSKURepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), skuRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true, LengthMM: int64Ptr(10),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVirtualSKUCannotHaveDimensions))
}

func TestUpdateSKU_allowsVirtualSKUWithoutDimensions(t *testing.T) {
	skuRepo := &stubSKURepo{}
	uc := apppms.NewUpdateSKUUseCase(newMockGormDB(t), skuRepo)

	sku, err := uc.Execute(context.Background(), apppms.UpdateSKUInput{
		ID: "sku-1", PriceMinorUnits: 100, CurrencyCode: "USD",
		Status: pmsDomain.SKUStatusActive, IsVirtual: true,
	})

	require.NoError(t, err)
	require.True(t, sku.IsVirtual)
}
