package pms

import (
	"context"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// writeSKUPriceHistory opens a new price-history interval for sku
// unconditionally -- used by CreateSKUUseCase, where there is no prior
// interval to compare against or close.
func writeSKUPriceHistory(ctx context.Context, repo pmsDomain.SKUPriceHistoryRepository, sku *pmsDomain.SKU, reason pmsDomain.PriceChangeReason) error {
	now := time.Now()
	return repo.Create(ctx, &pmsDomain.SKUPriceHistory{
		TenantID:            tenant.IDFromContext(ctx),
		SKUID:               sku.ID,
		PriceMinorUnits:     sku.PriceMinorUnits,
		SalePriceMinorUnits: sku.SalePriceMinorUnits,
		CurrencyCode:        sku.CurrencyCode,
		ValidFrom:           now,
		Reason:              reason,
		CreatedAt:           now,
	})
}

// recordSKUPriceChangeIfNeeded closes sku's currently open price-history
// interval and opens a new one, but only when price or sale price actually
// differ from before (oldPriceMinorUnits/oldSalePriceMinorUnits, captured
// prior to overwriting sku's fields) -- an update that leaves both
// untouched (e.g. renaming SKUCode) must not fabricate a price-change
// event. Called from inside the same tenant.WithTenant transaction as the
// SKU write itself, so the audit trail is atomic with the change it
// records.
func recordSKUPriceChangeIfNeeded(
	ctx context.Context,
	repo pmsDomain.SKUPriceHistoryRepository,
	sku *pmsDomain.SKU,
	oldPriceMinorUnits int64,
	oldSalePriceMinorUnits *int64,
	reason pmsDomain.PriceChangeReason,
) error {
	if sku.PriceMinorUnits == oldPriceMinorUnits && int64PtrEqual(sku.SalePriceMinorUnits, oldSalePriceMinorUnits) {
		return nil
	}
	if err := repo.CloseOpen(ctx, sku.ID, time.Now()); err != nil {
		return err
	}
	return writeSKUPriceHistory(ctx, repo, sku, reason)
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
