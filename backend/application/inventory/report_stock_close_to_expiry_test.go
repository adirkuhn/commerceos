package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/locale"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// readerCall is one ListExpiringStock call: the band's dates and the
// window whose acknowledged batches it skips.
type readerCall struct {
	from, to   string
	windowDays int
}

type stubExpiringStockReader struct {
	calls  []readerCall
	byFrom map[string][]invDomain.ExpiringStockLine
	err    error
}

func (s *stubExpiringStockReader) ListExpiringStock(_ context.Context, from, to time.Time, windowDays int) ([]invDomain.ExpiringStockLine, error) {
	s.calls = append(s.calls, readerCall{from.Format("2006-01-02"), to.Format("2006-01-02"), windowDays})
	return s.byFrom[from.Format("2006-01-02")], s.err
}

func fixedClock(s string) func() time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return func() time.Time { return t }
}

func expirySettings(windows ...int) settingsDomain.TenantSettings {
	return settingsDomain.TenantSettings{
		TenantID: "tenant-1", Timezone: "Europe/Amsterdam", DateFormat: settingsDomain.DateFormatDayMonthYear, Language: locale.Dutch,
		ExpiryAlert: settingsDomain.ExpiryAlert{Enabled: true, Windows: windows},
	}
}

func TestReportStockCloseToExpiry_raisesOneEventPerNonEmptyBand(t *testing.T) {
	sale := int64(399)
	reader := &stubExpiringStockReader{byFrom: map[string][]invDomain.ExpiringStockLine{
		// only the 30-day band (8-30 days left) has stock
		"2026-10-19": {{SKUID: "sku-1", SKUCode: "PQ-1", ProductName: "Pão de queijo", ImageURL: "https://cdn/pq.jpg",
			ExpiryDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), Quantity: 7,
			PriceMinorUnits: 499, SalePriceMinorUnits: &sale, CurrencyCode: "EUR"}},
	}}
	outbox := &eventrecordtest.Outbox{}
	// 23:30 UTC on the 10th is already the 11th in Amsterdam.
	uc := invapp.NewReportStockCloseToExpiryUseCase(newMockGormDB(t), reader, outbox, fixedClock("2026-10-10T23:30:00Z"))

	err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), expirySettings(30, 7))

	require.NoError(t, err)
	assert.Equal(t, []readerCall{{"2026-10-19", "2026-11-10", 30}, {"2026-10-11", "2026-10-18", 7}}, reader.calls)
	require.Len(t, outbox.Events, 1)
	ev := outbox.Events[0]
	assert.Equal(t, "inventory.stock.close_to_expiry_reported", ev.Subject)
	assert.Equal(t, "tenant-1", ev.Payload["tenant_id"])
	assert.Equal(t, float64(30), ev.Payload["window_days"])
	assert.Equal(t, float64(8), ev.Payload["min_days_left"])
	assert.Equal(t, float64(1), ev.Payload["item_count"])
	assert.Equal(t, []any{map[string]any{
		"sku_id": "sku-1", "product_name": "Pão de queijo", "sku": "PQ-1", "image_url": "https://cdn/pq.jpg",
		"quantity": float64(7), "expiry_date": "30/10/2026", "days_left": float64(19),
		"price": "€ 4,99", "sale_price": "€ 3,99",
	}}, ev.Payload["items"])
}

func TestReportStockCloseToExpiry_inactiveSaleShowsNoSalePrice(t *testing.T) {
	sale, ended := int64(399), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	reader := &stubExpiringStockReader{byFrom: map[string][]invDomain.ExpiringStockLine{
		"2026-10-10": {{SKUCode: "PQ-1", ExpiryDate: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), Quantity: 1,
			PriceMinorUnits: 499, SalePriceMinorUnits: &sale, SaleEndDate: &ended, CurrencyCode: "EUR"}},
	}}
	outbox := &eventrecordtest.Outbox{}
	uc := invapp.NewReportStockCloseToExpiryUseCase(newMockGormDB(t), reader, outbox, fixedClock("2026-10-10T08:00:00Z"))

	require.NoError(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), expirySettings(7)))

	require.Len(t, outbox.Events, 1)
	item := outbox.Events[0].Payload["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "", item["sale_price"])
	assert.Equal(t, float64(2), item["days_left"])
}

func TestReportStockCloseToExpiry_disabledDoesNothing(t *testing.T) {
	reader := &stubExpiringStockReader{}
	s := expirySettings(30)
	s.ExpiryAlert.Enabled = false
	uc := invapp.NewReportStockCloseToExpiryUseCase(newMockGormDBN(t, 0), reader, &eventrecordtest.Outbox{}, fixedClock("2026-10-10T08:00:00Z"))

	require.NoError(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), s))
	assert.Empty(t, reader.calls)
}

func TestReportStockCloseToExpiry_unknownTimezoneFallsBackToUTC(t *testing.T) {
	reader := &stubExpiringStockReader{}
	s := expirySettings(7)
	s.Timezone = ""
	uc := invapp.NewReportStockCloseToExpiryUseCase(newMockGormDB(t), reader, &eventrecordtest.Outbox{}, fixedClock("2026-10-10T23:30:00Z"))

	require.NoError(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), s))
	assert.Equal(t, []readerCall{{"2026-10-10", "2026-10-17", 7}}, reader.calls)
}

func TestReportStockCloseToExpiry_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	line := []invDomain.ExpiringStockLine{{SKUCode: "X", ExpiryDate: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), Quantity: 1, CurrencyCode: "EUR"}}

	uc := invapp.NewReportStockCloseToExpiryUseCase(newMockGormDB(t), &stubExpiringStockReader{err: boom}, &eventrecordtest.Outbox{}, fixedClock("2026-10-10T08:00:00Z"))
	assert.ErrorIs(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), expirySettings(7)), boom)

	uc = invapp.NewReportStockCloseToExpiryUseCase(newMockGormDB(t), &stubExpiringStockReader{byFrom: map[string][]invDomain.ExpiringStockLine{"2026-10-10": line}},
		&eventrecordtest.Outbox{Err: boom}, fixedClock("2026-10-10T08:00:00Z"))
	assert.ErrorIs(t, uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), expirySettings(7)), boom)
}
