package inventory

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SKUAvailability is what a SKU can really be sold from right now -- for
// quoting, not channel sync.
type SKUAvailability struct {
	// Available is every unit not sitting in an expired lot. Unlike
	// StockLevel.Syncable it never applies the tenant's FEFO sync rules,
	// which under-report stock on purpose for channels.
	Available int64
	// EarliestExpiry is the soonest expiry among not-yet-expired lots still
	// holding stock; nil when no such lot carries an expiry date.
	EarliestExpiry *time.Time
}

// GetSKUAvailabilityUseCase computes SKUAvailability for a known set of
// SKUs with two bulk queries, same shape as GetStockLevelsUseCase.
type GetSKUAvailabilityUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
}

func NewGetSKUAvailabilityUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository) *GetSKUAvailabilityUseCase {
	return &GetSKUAvailabilityUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo}
}

// Execute returns an entry for every requested SKU id (zero value when it
// has no stock history).
func (uc *GetSKUAvailabilityUseCase) Execute(ctx context.Context, skuIDs []string) (map[string]SKUAvailability, error) {
	result := make(map[string]SKUAvailability, len(skuIDs))
	if len(skuIDs) == 0 {
		return result, nil
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKUs(ctx, skuIDs)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKUs(ctx, skuIDs)
		if err != nil {
			return err
		}
		movementsBySKU := make(map[string][]invDomain.StockMovement, len(skuIDs))
		for _, m := range movements {
			movementsBySKU[m.SKUID] = append(movementsBySKU[m.SKUID], m)
		}
		batchesBySKU := make(map[string][]invDomain.Batch, len(skuIDs))
		for _, b := range batches {
			batchesBySKU[b.SKUID] = append(batchesBySKU[b.SKUID], b)
		}
		now := time.Now()
		for _, id := range skuIDs {
			result[id] = computeAvailability(id, movementsBySKU[id], batchesBySKU[id], now)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSKUAvailability: %w", err)
	}
	return result, nil
}

// computeAvailability reuses computeStockLevel's flat rule (rules off) for
// the quantity, so "not in an expired lot" can't drift between the two.
func computeAvailability(skuID string, movements []invDomain.StockMovement, batches []invDomain.Batch, now time.Time) SKUAvailability {
	availability := SKUAvailability{Available: computeStockLevel(skuID, movements, batches, false).Syncable}

	qtyByBatch := make(map[string]int64, len(batches))
	for _, m := range movements {
		if m.BatchID != nil {
			qtyByBatch[*m.BatchID] += m.QuantityDelta
		}
	}
	for _, b := range batches {
		if b.ExpiryDate == nil || !b.ExpiryDate.After(now) || qtyByBatch[b.ID] <= 0 {
			continue
		}
		if availability.EarliestExpiry == nil || b.ExpiryDate.Before(*availability.EarliestExpiry) {
			expiry := *b.ExpiryDate
			availability.EarliestExpiry = &expiry
		}
	}
	return availability
}
