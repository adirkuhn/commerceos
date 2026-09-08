package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListSKUCurrencyPricesUseCase_Execute(t *testing.T) {
	repo := &stubSKUCurrencyPriceRepo{byKey: map[string]pmsDomain.SKUCurrencyPrice{
		"sku-1|USD": {SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1200},
	}}
	uc := apppms.NewListSKUCurrencyPricesUseCase(newMockGormDBN(t, 1), repo)

	prices, err := uc.Execute(context.Background(), "sku-1")
	require.NoError(t, err)
	require.Len(t, prices, 1)
	require.Equal(t, "USD", prices[0].CurrencyCode)
}
