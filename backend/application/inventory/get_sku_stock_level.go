package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// StockLevel is a SKU's current computed stock, derived from its
// StockMovement ledger — never stored directly.
type StockLevel struct {
	SKUID string
	// Total is the sum of every movement, regardless of batch expiry.
	Total int64
	// Syncable is what should be reported to external channels: the
	// default rule sums un-lotted movements plus movements against
	// batches that aren't expired as of now. This is the one stock rule
	// in place today — a pluggable rules engine (excluding reserved
	// stock, specific warehouses, etc.) is future work, not built yet.
	Syncable int64
}

type GetSKUStockLevelUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	batchRepo    invDomain.BatchRepository
}

func NewGetSKUStockLevelUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, batchRepo invDomain.BatchRepository) *GetSKUStockLevelUseCase {
	return &GetSKUStockLevelUseCase{db: db, movementRepo: movementRepo, batchRepo: batchRepo}
}

func (uc *GetSKUStockLevelUseCase) Execute(ctx context.Context, skuID string) (*StockLevel, error) {
	level := StockLevel{SKUID: skuID}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		batches, err := uc.batchRepo.ListBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		expiryByBatch := make(map[string]*time.Time, len(batches))
		for _, b := range batches {
			expiryByBatch[b.ID] = b.ExpiryDate
		}

		now := time.Now()
		for _, m := range movements {
			level.Total += m.QuantityDelta
			if m.BatchID == nil {
				level.Syncable += m.QuantityDelta
				continue
			}
			expiry := expiryByBatch[*m.BatchID]
			if expiry == nil || expiry.After(now) {
				level.Syncable += m.QuantityDelta
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSKUStockLevel: %w", err)
	}
	return &level, nil
}
