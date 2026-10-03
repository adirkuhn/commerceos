package pms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// enqueuePricePerKiloChangedIfNeeded raises pms.sku.price_per_kilo_changed
// in the caller's transaction when sku is a scale-sold weight pool whose
// rate differs from old -- its packages' prices derive from that rate (see
// RepriceWeightPackagesUseCase).
func enqueuePricePerKiloChangedIfNeeded(ctx context.Context, outboxRepo outbox.Repository, sku pmsDomain.SKU, old *int64) error {
	if !isScaleSoldWeightPool(sku) {
		return nil
	}
	if old != nil && *old == *sku.PricePerKiloMinorUnits {
		return nil
	}
	event := events.SKUPricePerKiloChanged{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
		TenantID: tenant.IDFromContext(ctx), SKUID: sku.ID,
		OldPricePerKiloMinorUnits: old, NewPricePerKiloMinorUnits: *sku.PricePerKiloMinorUnits,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
}
