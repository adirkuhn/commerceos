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

// batchFinder narrows inventory.BatchRepository to the lookup the
// acknowledgement use cases need (ISP).
type batchFinder interface {
	FindByID(ctx context.Context, id string) (*invDomain.Batch, error)
}

// BatchExpiryAcknowledgementInput names a batch and the close-to-expiry
// window it is (un)marked as seen for. By is who did it.
type BatchExpiryAcknowledgementInput struct {
	BatchID    string
	WindowDays int
	By         string
}

// AcknowledgeBatchExpiryUseCase is "mark as aware": the batch leaves that
// window's list and email report until it reaches the next window.
// Acknowledging twice changes nothing and raises no second event.
type AcknowledgeBatchExpiryUseCase struct {
	db       *gorm.DB
	settings tenantSettingsGetter
	batches  batchFinder
	acks     invDomain.ExpiryAcknowledgementRepository
	outbox   eventrecord.Enqueuer
	now      func() time.Time
}

func NewAcknowledgeBatchExpiryUseCase(db *gorm.DB, settings tenantSettingsGetter, batches batchFinder, acks invDomain.ExpiryAcknowledgementRepository, outbox eventrecord.Enqueuer, now func() time.Time) *AcknowledgeBatchExpiryUseCase {
	return &AcknowledgeBatchExpiryUseCase{db: db, settings: settings, batches: batches, acks: acks, outbox: outbox, now: now}
}

func (uc *AcknowledgeBatchExpiryUseCase) Execute(ctx context.Context, in BatchExpiryAcknowledgementInput) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.settings.Get(ctx)
		if err != nil {
			return err
		}
		band, ok := s.ExpiryAlert.ReviewBand(in.WindowDays)
		if !ok {
			return invDomain.ErrUnknownExpiryWindow
		}
		batch, err := uc.batches.FindByID(ctx, in.BatchID)
		if err != nil {
			return err
		}
		now := uc.now()
		ack, err := invDomain.NewExpiryAcknowledgement(*batch, band, tenantToday(now, s.Timezone), in.By, now)
		if err != nil {
			return err
		}
		created, err := uc.acks.Create(ctx, ack)
		if err != nil || !created {
			return err
		}
		return recordBatchExpiryAction(ctx, uc.outbox, events.BatchExpiryAcknowledged, *batch, in.WindowDays)
	})
	if err != nil {
		return fmt.Errorf("acknowledgeBatchExpiry: %w", err)
	}
	return nil
}

// UnacknowledgeBatchExpiryUseCase undoes "mark as aware" for one window.
// Undoing what is not there changes nothing and raises no event.
type UnacknowledgeBatchExpiryUseCase struct {
	db      *gorm.DB
	batches batchFinder
	acks    invDomain.ExpiryAcknowledgementRepository
	outbox  eventrecord.Enqueuer
}

func NewUnacknowledgeBatchExpiryUseCase(db *gorm.DB, batches batchFinder, acks invDomain.ExpiryAcknowledgementRepository, outbox eventrecord.Enqueuer) *UnacknowledgeBatchExpiryUseCase {
	return &UnacknowledgeBatchExpiryUseCase{db: db, batches: batches, acks: acks, outbox: outbox}
}

func (uc *UnacknowledgeBatchExpiryUseCase) Execute(ctx context.Context, in BatchExpiryAcknowledgementInput) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		batch, err := uc.batches.FindByID(ctx, in.BatchID)
		if err != nil {
			return err
		}
		deleted, err := uc.acks.Delete(ctx, in.BatchID, in.WindowDays)
		if err != nil || !deleted {
			return err
		}
		return recordBatchExpiryAction(ctx, uc.outbox, events.BatchExpiryUnacknowledged, *batch, in.WindowDays)
	})
	if err != nil {
		return fmt.Errorf("unacknowledgeBatchExpiry: %w", err)
	}
	return nil
}

func recordBatchExpiryAction(ctx context.Context, outbox eventrecord.Enqueuer, action events.ActionType, batch invDomain.Batch, windowDays int) error {
	return eventrecord.Record(ctx, outbox, action, batch.ID, eventrecord.Data{
		"window_days": windowDays, "sku_id": batch.SKUID, "warehouse_id": batch.WarehouseID, "lot_number": batch.LotNumber,
	})
}
