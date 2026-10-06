package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// CompleteReceivingUseCase marks a ReceivingRecord applied once a human has
// reviewed its discrepancies and booked stock for it (via
// inventory.ReceiveStockUseCase, called separately per line from the
// review UI -- see plan Phase 4). Discrepancies themselves stay
// informational; there's no per-discrepancy "resolved" state to track,
// completing the record is the single signal that review is done.
type CompleteReceivingUseCase struct {
	db      *gorm.DB
	recRepo procurementDomain.ReceivingRecordRepository
	outbox  eventrecord.Enqueuer
}

func NewCompleteReceivingUseCase(db *gorm.DB, recRepo procurementDomain.ReceivingRecordRepository, outbox eventrecord.Enqueuer) *CompleteReceivingUseCase {
	return &CompleteReceivingUseCase{db: db, recRepo: recRepo, outbox: outbox}
}

func (uc *CompleteReceivingUseCase) Execute(ctx context.Context, receivingRecordID string) (*procurementDomain.ReceivingRecord, error) {
	var rec *procurementDomain.ReceivingRecord
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		rec, err = uc.recRepo.FindByID(ctx, receivingRecordID)
		if err != nil {
			return err
		}
		rec.Status = procurementDomain.ReceivingRecordStatusApplied
		rec.UpdatedAt = time.Now()
		if err := uc.recRepo.Update(ctx, rec); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ReceivingRecordCompleted, rec.ID, eventrecord.Data{
			"purchase_order_id": rec.PurchaseOrderID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("completeReceiving: %w", err)
	}
	return rec, nil
}
