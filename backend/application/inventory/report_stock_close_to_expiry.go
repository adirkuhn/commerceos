package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ReportStockCloseToExpiryUseCase is one tenant's daily stock-close-to-
// expiry report: for each band of its chosen windows (see
// settingsDomain.ExpiryAlert.Bands) with stock in it, it raises a
// StockCloseToExpiryReported event. Who hears about it is up to the
// event's consumers -- the Message Center rule emails it, notifications
// puts it in the bell. Values are pre-formatted in the tenant's language,
// date format and time zone, so templates just print them.
type ReportStockCloseToExpiryUseCase struct {
	db     *gorm.DB
	reader invDomain.ExpiringStockReader
	outbox eventrecord.Enqueuer
	now    func() time.Time
}

func NewReportStockCloseToExpiryUseCase(db *gorm.DB, reader invDomain.ExpiringStockReader, outbox eventrecord.Enqueuer, now func() time.Time) *ReportStockCloseToExpiryUseCase {
	return &ReportStockCloseToExpiryUseCase{db: db, reader: reader, outbox: outbox, now: now}
}

// Execute reports for the tenant in ctx, whose settings are s.
func (uc *ReportStockCloseToExpiryUseCase) Execute(ctx context.Context, s settingsDomain.TenantSettings) error {
	bands := s.ExpiryAlert.Bands()
	if len(bands) == 0 {
		return nil
	}
	today := tenantToday(uc.now(), s.Timezone)

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, band := range bands {
			lines, err := uc.reader.ListExpiringStock(ctx, today.AddDate(0, 0, band.MinDaysLeft), today.AddDate(0, 0, band.WindowDays))
			if err != nil {
				return err
			}
			if len(lines) == 0 {
				continue
			}
			if err := uc.enqueue(ctx, s, band, reportItems(lines, s, today)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reportStockCloseToExpiry: %w", err)
	}
	return nil
}

func (uc *ReportStockCloseToExpiryUseCase) enqueue(ctx context.Context, s settingsDomain.TenantSettings, band settingsDomain.ExpiryBand, items []events.StockCloseToExpiryItem) error {
	event := events.StockCloseToExpiryReported{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: uc.now().UTC()},
		TenantID: s.TenantID, WindowDays: band.WindowDays, MinDaysLeft: band.MinDaysLeft,
		ItemCount: len(items), Items: items,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return uc.outbox.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}

func reportItems(lines []invDomain.ExpiringStockLine, s settingsDomain.TenantSettings, today time.Time) []events.StockCloseToExpiryItem {
	items := make([]events.StockCloseToExpiryItem, len(lines))
	for i, l := range lines {
		salePrice := ""
		if sale := l.ActiveSalePrice(today); sale != nil {
			salePrice = s.Language.FormatMoney(*sale, l.CurrencyCode)
		}
		items[i] = events.StockCloseToExpiryItem{
			SKUID: l.SKUID, ProductName: l.ProductName, SKU: l.SKUCode, ImageURL: l.ImageURL,
			Quantity: l.Quantity, ExpiryDate: s.DateFormat.Format(l.ExpiryDate), DaysLeft: l.DaysLeft(today),
			Price: s.Language.FormatMoney(l.PriceMinorUnits, l.CurrencyCode), SalePrice: salePrice,
		}
	}
	return items
}

// tenantToday is the calendar date it is now in the tenant's time zone
// (UTC when unset or unknown), at midnight UTC so date math stays exact.
func tenantToday(now time.Time, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if timezone == "" || err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}
