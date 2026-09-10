package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ExpireBatchUseCase flips a single batch's Status to BatchStatusExpired.
// Called by cmd/scheduler once per row returned by
// BatchRepository.ListToExpire, under that batch's own tenant context (see
// runExpiredSalePrices's comment in cmd/scheduler for why cross-tenant
// polling still dispatches per-item work under the real tenant identity).
// It never touches the StockMovement ledger or stock quantity (see
// domain/inventory.BatchStatusExpired's doc comment) -- but it does raise
// inventory.stock.changed, same as every other stock-affecting use case
// (see update_batch.go), because computeStockLevel's live ExpiryDate-vs-now()
// check means this SKU's Syncable quantity just changed even though no
// StockMovement row did. Without it, an auto-expired batch's lower quantity
// is correct internally but never reaches channels until some unrelated
// event happens to touch the same SKU.
type ExpireBatchUseCase struct {
	db         *gorm.DB
	batchRepo  invDomain.BatchRepository
	outboxRepo outbox.Repository
}

func NewExpireBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, outboxRepo outbox.Repository) *ExpireBatchUseCase {
	return &ExpireBatchUseCase{db: db, batchRepo: batchRepo, outboxRepo: outboxRepo}
}

func (uc *ExpireBatchUseCase) Execute(ctx context.Context, batchID, skuID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.batchRepo.MarkExpired(ctx, batchID); err != nil {
			return err
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, skuID)
	})
	if err != nil {
		return fmt.Errorf("expireBatch: %w", err)
	}
	return nil
}
