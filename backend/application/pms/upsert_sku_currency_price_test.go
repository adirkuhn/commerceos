package pms_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUpsertSKUCurrencyPriceUseCase_Execute(t *testing.T) {
	t.Run("rejects an invalid currency code", func(t *testing.T) {
		uc := apppms.NewUpsertSKUCurrencyPriceUseCase(newMockGormDBN(t, 0), &stubSKURepo{}, &stubSKUCurrencyPriceRepo{}, &eventrecordtest.Outbox{})

		_, err := uc.Execute(context.Background(), apppms.UpsertSKUCurrencyPriceInput{SKUID: "sku-1", CurrencyCode: "usd", PriceMinorUnits: 1000})
		require.ErrorIs(t, err, fxDomain.ErrInvalidCurrencyCode)
	})

	t.Run("rejects a sale price exceeding the regular price", func(t *testing.T) {
		uc := apppms.NewUpsertSKUCurrencyPriceUseCase(newMockGormDBN(t, 0), &stubSKURepo{}, &stubSKUCurrencyPriceRepo{}, &eventrecordtest.Outbox{})

		_, err := uc.Execute(context.Background(), apppms.UpsertSKUCurrencyPriceInput{
			SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1000, SalePriceMinorUnits: int64Ptr(1200),
		})
		require.ErrorIs(t, err, pmsDomain.ErrSalePriceExceedsPrice)
	})

	t.Run("rejects a sale end date with no sale price", func(t *testing.T) {
		uc := apppms.NewUpsertSKUCurrencyPriceUseCase(newMockGormDBN(t, 0), &stubSKURepo{}, &stubSKUCurrencyPriceRepo{}, &eventrecordtest.Outbox{})
		end := time.Now()

		_, err := uc.Execute(context.Background(), apppms.UpsertSKUCurrencyPriceInput{
			SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1000, SaleEndDate: &end,
		})
		require.ErrorIs(t, err, pmsDomain.ErrSaleEndDateRequiresSalePrice)
	})

	t.Run("upserts a valid override", func(t *testing.T) {
		skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
		repo := &stubSKUCurrencyPriceRepo{}
		uc := apppms.NewUpsertSKUCurrencyPriceUseCase(newMockGormDBN(t, 1), skuRepo, repo, &eventrecordtest.Outbox{})

		price, err := uc.Execute(context.Background(), apppms.UpsertSKUCurrencyPriceInput{
			SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1200, SalePriceMinorUnits: int64Ptr(999),
		})
		require.NoError(t, err)
		require.Equal(t, "USD", price.CurrencyCode)
		require.Equal(t, int64(1200), price.PriceMinorUnits)
		require.Equal(t, "sku-1", repo.upserted.SKUID)
	})

	t.Run("propagates sku not found", func(t *testing.T) {
		skuRepo := &stubSKURepo{findByIDErr: pmsDomain.ErrSKUNotFound}
		uc := apppms.NewUpsertSKUCurrencyPriceUseCase(newMockGormDBN(t, 1), skuRepo, &stubSKUCurrencyPriceRepo{}, &eventrecordtest.Outbox{})

		_, err := uc.Execute(context.Background(), apppms.UpsertSKUCurrencyPriceInput{SKUID: "missing", CurrencyCode: "USD", PriceMinorUnits: 1000})
		require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
	})
}
