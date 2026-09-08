package fx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appfx "github.com/adirkuhn/commerceos/application/fx"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func TestUpsertExchangeRateUseCase_Execute(t *testing.T) {
	t.Run("rejects an invalid currency code", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{}
		uc := appfx.NewUpsertExchangeRateUseCase(newMockGormDB(t, 0), repo)

		_, err := uc.Execute(context.Background(), appfx.UpsertExchangeRateInput{CurrencyCode: "eur", RateBasisPoints: 10000})
		require.ErrorIs(t, err, fxDomain.ErrInvalidCurrencyCode)
	})

	t.Run("rejects a zero or negative rate", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{}
		uc := appfx.NewUpsertExchangeRateUseCase(newMockGormDB(t, 0), repo)

		_, err := uc.Execute(context.Background(), appfx.UpsertExchangeRateInput{CurrencyCode: "USD", RateBasisPoints: 0})
		require.ErrorIs(t, err, fxDomain.ErrInvalidRate)
	})

	t.Run("upserts a valid rate", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{}
		uc := appfx.NewUpsertExchangeRateUseCase(newMockGormDB(t, 1), repo)

		rate, err := uc.Execute(context.Background(), appfx.UpsertExchangeRateInput{CurrencyCode: "USD", RateBasisPoints: 10876})
		require.NoError(t, err)
		require.Equal(t, "USD", rate.CurrencyCode)
		require.Equal(t, int64(10876), rate.RateBasisPoints)
		require.Equal(t, "USD", repo.upserted.CurrencyCode)
	})
}
