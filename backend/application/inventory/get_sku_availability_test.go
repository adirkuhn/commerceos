package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSKUAvailability_countsUnexpiredStockAndSoonestExpiryHoldingStock(t *testing.T) {
	now := time.Now()
	expired := now.AddDate(0, 0, -2)
	soonDepleted := now.AddDate(0, 0, 5)
	soon := now.AddDate(0, 0, 10)
	later := now.AddDate(0, 1, 0)
	batches := []invDomain.Batch{
		{ID: "b-expired", SKUID: "sku-1", ExpiryDate: &expired},
		{ID: "b-depleted", SKUID: "sku-1", ExpiryDate: &soonDepleted},
		{ID: "b-soon", SKUID: "sku-1", ExpiryDate: &soon},
		{ID: "b-later", SKUID: "sku-1", ExpiryDate: &later},
	}
	movements := []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("b-expired"), QuantityDelta: 4},
		{SKUID: "sku-1", BatchID: strPtr("b-depleted"), QuantityDelta: 3},
		{SKUID: "sku-1", BatchID: strPtr("b-depleted"), QuantityDelta: -3},
		{SKUID: "sku-1", BatchID: strPtr("b-soon"), QuantityDelta: 6},
		{SKUID: "sku-1", BatchID: strPtr("b-later"), QuantityDelta: 10},
		{SKUID: "sku-1", QuantityDelta: 2},
	}
	uc := appinv.NewGetSKUAvailabilityUseCase(newMockGormDB(t),
		&stubStockMovementRepo{bySKUs: movements}, &stubBatchRepo{batches: batches})

	got, err := uc.Execute(context.Background(), []string{"sku-1"})

	require.NoError(t, err)
	assert.Equal(t, int64(18), got["sku-1"].Available, "expired lot excluded, un-lotted stock included")
	require.NotNil(t, got["sku-1"].EarliestExpiry)
	assert.True(t, soon.Equal(*got["sku-1"].EarliestExpiry), "depleted and expired lots are skipped")
}

// Every unexpired lot counts, not just the FEFO lot channel sync reports
// when the tenant's sync rules are on -- a B2B quote sells all of it.
func TestGetSKUAvailability_countsEveryUnexpiredLot(t *testing.T) {
	soon := time.Now().AddDate(0, 0, 10)
	later := time.Now().AddDate(0, 1, 0)
	batches := []invDomain.Batch{
		{ID: "b-soon", SKUID: "sku-1", ExpiryDate: &soon},
		{ID: "b-later", SKUID: "sku-1", ExpiryDate: &later},
	}
	movements := []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: strPtr("b-soon"), QuantityDelta: 6},
		{SKUID: "sku-1", BatchID: strPtr("b-later"), QuantityDelta: 10},
	}
	uc := appinv.NewGetSKUAvailabilityUseCase(newMockGormDB(t),
		&stubStockMovementRepo{bySKUs: movements}, &stubBatchRepo{batches: batches})

	got, err := uc.Execute(context.Background(), []string{"sku-1"})

	require.NoError(t, err)
	assert.Equal(t, int64(16), got["sku-1"].Available)
}

func TestGetSKUAvailability_skuWithoutHistoryHasNothing(t *testing.T) {
	uc := appinv.NewGetSKUAvailabilityUseCase(newMockGormDB(t),
		&stubStockMovementRepo{}, &stubBatchRepo{})

	got, err := uc.Execute(context.Background(), []string{"sku-1"})

	require.NoError(t, err)
	assert.Equal(t, appinv.SKUAvailability{}, got["sku-1"])
}

func TestGetSKUAvailability_nothingRequestedTouchesNothing(t *testing.T) {
	got, err := appinv.NewGetSKUAvailabilityUseCase(nil, &stubStockMovementRepo{}, &stubBatchRepo{}).
		Execute(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGetSKUAvailability_wrapsRepoError(t *testing.T) {
	uc := appinv.NewGetSKUAvailabilityUseCase(newMockGormDB(t),
		&stubStockMovementRepo{bySKUsErr: errors.New("boom")}, &stubBatchRepo{})

	_, err := uc.Execute(context.Background(), []string{"sku-1"})

	assert.ErrorContains(t, err, "getSKUAvailability")
}
