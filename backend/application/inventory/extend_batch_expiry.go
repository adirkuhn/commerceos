package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ExtendBatchExpiryUseCase is the review queue's "Extend" action -- pushes
// a batch's expiry date out and un-expires it (Status back to Active).
type ExtendBatchExpiryUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewExtendBatchExpiryUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *ExtendBatchExpiryUseCase {
	return &ExtendBatchExpiryUseCase{db: db, batchRepo: batchRepo}
}

func (uc *ExtendBatchExpiryUseCase) Execute(ctx context.Context, batchID string, newExpiryDate time.Time) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.batchRepo.ExtendExpiry(ctx, batchID, newExpiryDate)
	})
	if err != nil {
		return fmt.Errorf("extendBatchExpiry: %w", err)
	}
	return nil
}
