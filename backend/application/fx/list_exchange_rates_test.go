package fx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appfx "github.com/adirkuhn/commerceos/application/fx"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func TestListExchangeRatesUseCase_Execute(t *testing.T) {
	repo := &fakeExchangeRateRepo{byCurrency: map[string]fxDomain.ExchangeRate{
		"USD": {CurrencyCode: "USD", RateBasisPoints: 10876},
	}}
	uc := appfx.NewListExchangeRatesUseCase(newMockGormDB(t, 1), repo)

	rates, err := uc.Execute(context.Background())
	require.NoError(t, err)
	require.Len(t, rates, 1)
	require.Equal(t, "USD", rates[0].CurrencyCode)
}
