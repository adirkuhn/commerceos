package inventory_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/settings"
)

func TestNewExpiryAcknowledgement_ForTheBandTheBatchIsIn(t *testing.T) {
	expiry := day("2026-11-30")
	batch := inventory.Batch{ID: "b-1", TenantID: "t-1", ExpiryDate: &expiry}
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	ack, err := inventory.NewExpiryAcknowledgement(batch, settings.ExpiryBand{WindowDays: 90, MinDaysLeft: 31}, day("2026-10-12"), "u-1", at)

	require.NoError(t, err)
	assert.Equal(t, inventory.ExpiryAcknowledgement{
		TenantID: "t-1", BatchID: "b-1", WindowDays: 90, AcknowledgedBy: "u-1", AcknowledgedAt: at,
	}, ack)
}

func TestNewExpiryAcknowledgement_RejectsABatchOutsideTheBand(t *testing.T) {
	expiry := day("2026-10-30") // 18 days left: in the 30-day band, not the 90-day one
	batch := inventory.Batch{ID: "b-1", ExpiryDate: &expiry}

	_, err := inventory.NewExpiryAcknowledgement(batch, settings.ExpiryBand{WindowDays: 90, MinDaysLeft: 31}, day("2026-10-12"), "u-1", time.Now())

	assert.ErrorIs(t, err, inventory.ErrBatchNotInExpiryWindow)
}

func TestNewExpiryAcknowledgement_RejectsABatchWithoutExpiryDate(t *testing.T) {
	_, err := inventory.NewExpiryAcknowledgement(inventory.Batch{ID: "b-1"}, settings.ExpiryBand{WindowDays: 90}, day("2026-10-12"), "u-1", time.Now())

	assert.ErrorIs(t, err, inventory.ErrBatchNotInExpiryWindow)
}

func TestDaysUntil(t *testing.T) {
	assert.Equal(t, 10, inventory.DaysUntil(day("2026-10-20"), day("2026-10-10")))
	assert.Equal(t, 0, inventory.DaysUntil(day("2026-10-10"), day("2026-10-10")))
	assert.Equal(t, -1, inventory.DaysUntil(day("2026-10-09"), day("2026-10-10")))
}

func TestCloseToExpiryBatch_ActiveSalePrice(t *testing.T) {
	sale := int64(399)
	ended := day("2026-10-11")
	assert.Equal(t, &sale, inventory.CloseToExpiryBatch{SalePriceMinorUnits: &sale}.ActiveSalePrice(day("2026-10-12")))
	assert.Nil(t, inventory.CloseToExpiryBatch{SalePriceMinorUnits: &sale, SaleEndDate: &ended}.ActiveSalePrice(day("2026-10-12")))
	assert.Nil(t, inventory.CloseToExpiryBatch{}.ActiveSalePrice(day("2026-10-12")))
}
