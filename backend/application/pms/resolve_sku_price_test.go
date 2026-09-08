package pms_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestResolveSKUPrice(t *testing.T) {
	sku := pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 1000, SalePriceMinorUnits: int64Ptr(800), CurrencyCode: "EUR"}

	t.Run("empty channel currency passes the SKU's own price through unchanged", func(t *testing.T) {
		overrideRepo := &stubSKUCurrencyPriceRepo{}
		fxRepo := &stubExchangeRateRepoPMS{}

		resolved, err := apppms.ResolveSKUPrice(context.Background(), sku, "", overrideRepo, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1000), resolved.PriceMinorUnits)
		require.Equal(t, int64(800), *resolved.SalePriceMinorUnits)
		require.Equal(t, "EUR", resolved.CurrencyCode)
	})

	t.Run("channel currency matching the SKU's own currency needs no conversion", func(t *testing.T) {
		overrideRepo := &stubSKUCurrencyPriceRepo{}
		fxRepo := &stubExchangeRateRepoPMS{}

		resolved, err := apppms.ResolveSKUPrice(context.Background(), sku, "EUR", overrideRepo, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1000), resolved.PriceMinorUnits)
	})

	t.Run("a manual override wins over fx conversion", func(t *testing.T) {
		overrideRepo := &stubSKUCurrencyPriceRepo{byKey: map[string]pmsDomain.SKUCurrencyPrice{
			"sku-1|USD": {SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1200, SalePriceMinorUnits: int64Ptr(999)},
		}}
		fxRepo := &stubExchangeRateRepoPMS{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 20000}}}

		resolved, err := apppms.ResolveSKUPrice(context.Background(), sku, "USD", overrideRepo, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1200), resolved.PriceMinorUnits)
		require.Equal(t, int64(999), *resolved.SalePriceMinorUnits)
		require.Equal(t, "USD", resolved.CurrencyCode)
	})

	t.Run("an expired override sale price is treated as no sale", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		overrideRepo := &stubSKUCurrencyPriceRepo{byKey: map[string]pmsDomain.SKUCurrencyPrice{
			"sku-1|USD": {SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1200, SalePriceMinorUnits: int64Ptr(999), SaleEndDate: &past},
		}}
		fxRepo := &stubExchangeRateRepoPMS{}

		resolved, err := apppms.ResolveSKUPrice(context.Background(), sku, "USD", overrideRepo, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1200), resolved.PriceMinorUnits)
		require.Nil(t, resolved.SalePriceMinorUnits)
	})

	t.Run("no override, converts via the configured fx rate", func(t *testing.T) {
		overrideRepo := &stubSKUCurrencyPriceRepo{}
		fxRepo := &stubExchangeRateRepoPMS{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 10876}}}

		resolved, err := apppms.ResolveSKUPrice(context.Background(), sku, "USD", overrideRepo, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1088), resolved.PriceMinorUnits)     // 1000 * 1.0876, rounded
		require.Equal(t, int64(870), *resolved.SalePriceMinorUnits) // 800 * 1.0876, rounded
		require.Equal(t, "USD", resolved.CurrencyCode)
	})

	t.Run("no override and no fx rate is a hard error", func(t *testing.T) {
		overrideRepo := &stubSKUCurrencyPriceRepo{}
		fxRepo := &stubExchangeRateRepoPMS{}

		_, err := apppms.ResolveSKUPrice(context.Background(), sku, "USD", overrideRepo, fxRepo)
		require.ErrorIs(t, err, pmsDomain.ErrExchangeRateRequired)
	})
}
