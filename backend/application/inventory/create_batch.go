package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	outbox    eventrecord.Enqueuer
}

func NewCreateBatchUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, outbox eventrecord.Enqueuer) *CreateBatchUseCase {
	return &CreateBatchUseCase{db: db, batchRepo: batchRepo, outbox: outbox}
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
			Status:      invDomain.BatchStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := uc.batchRepo.Create(ctx, &batch); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BatchCreated, batch.ID, eventrecord.Data{
			"sku_id": batch.SKUID, "warehouse_id": batch.WarehouseID, "lot_number": batch.LotNumber, "expiry_date": batch.ExpiryDate,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createBatch: %w", err)
	}
	return &batch, nil
}
