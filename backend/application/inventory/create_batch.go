package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateBatchInput struct {
	SKUID       string
	WarehouseID string
	LotNumber   string
	ExpiryDate  *time.Time
	ReceivedAt  time.Time // zero value means "now"
}

type CreateBatchUseCase struct {
	db        *gorm.DB
	batchRepo invDomain.BatchRepository
}

func NewCreateBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository) *CreateBatchUseCase {
	return &CreateBatchUseCase{db: db, batchRepo: batchRepo}
}

func (uc *CreateBatchUseCase) Execute(ctx context.Context, in CreateBatchInput) (*invDomain.Batch, error) {
	var batch invDomain.Batch
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		receivedAt := in.ReceivedAt
		if receivedAt.IsZero() {
			receivedAt = now
		}
		batch = invDomain.Batch{
			TenantID:    tenant.IDFromContext(ctx),
			SKUID:       in.SKUID,
			WarehouseID: in.WarehouseID,
			LotNumber:   in.LotNumber,
			ExpiryDate:  in.ExpiryDate,
			ReceivedAt:  receivedAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		return uc.batchRepo.Create(ctx, &batch)
	})
	if err != nil {
		return nil, fmt.Errorf("createBatch: %w", err)
	}
	return &batch, nil
}
