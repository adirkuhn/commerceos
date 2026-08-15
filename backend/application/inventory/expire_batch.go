package inventory

import (
	"context"
	"fmt"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ExpireBatchUseCase flips a single batch's Status to BatchStatusExpired.
// Called by cmd/scheduler once per row returned by
// BatchRepository.ListToExpire, under that batch's own tenant context (see
// runExpiredSalePrices's comment in cmd/scheduler for why cross-tenant
// polling still dispatches per-item work under the real tenant identity).
// Purely a status flag flip -- it never touches the StockMovement ledger or
// stock quantity (see domain/inventory.BatchStatusExpired's doc comment).
type ExpireBatchUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewExpireBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *ExpireBatchUseCase {
	return &ExpireBatchUseCase{db: db, batchRepo: batchRepo}
}

func (uc *ExpireBatchUseCase) Execute(ctx context.Context, batchID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.batchRepo.MarkExpired(ctx, batchID)
	})
	if err != nil {
		return fmt.Errorf("expireBatch: %w", err)
	}
	return nil
}
