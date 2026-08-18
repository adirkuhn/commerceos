package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type MarkDamagedOrLostInput struct {
	SKUID       string
	WarehouseID string
	BatchID     *string
	Quantity    int64 // always positive — the amount written off
	Type        invDomain.MovementType
	Reason      string
	OccurredAt  time.Time
	// DiscardedBy is who/what performed the write-off -- a user ID for the
	// inventory page's Damaged/Lost button or the scanner Discard app, or
	// a booking scan's ScannedBy for a procurement replay. Stored on the
	// DiscardRecord this call always produces alongside the movement (see
	// that struct's doc comment) -- every caller feeds the same unified
	// discarded/damaged/lost list.
	DiscardedBy string
	// PhotoURL is nil unless the caller already uploaded one (only
	// DiscardBatchUseCase's flow ever does).
	PhotoURL *string
}

type MarkDamagedOrLostUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	discardRepo  invDomain.DiscardRecordRepository
	outboxRepo   outbox.Repository
}

func NewMarkDamagedOrLostUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, discardRepo invDomain.DiscardRecordRepository, outboxRepo outbox.Repository) *MarkDamagedOrLostUseCase {
	return &MarkDamagedOrLostUseCase{db: db, movementRepo: movementRepo, discardRepo: discardRepo, outboxRepo: outboxRepo}
}

// Execute records a write-off — stock that's gone (damaged or lost) and
// won't be sold. Type must be MovementTypeDamaged or MovementTypeLost;
// anything else is rejected rather than silently written as some other
// kind of movement. Every call writes both the StockMovement AND a
// DiscardRecord (same transaction) -- this is the ONE place that happens,
// so every caller (the inventory page's Damaged/Lost button,
// DiscardBatchUseCase's expiry/scanner discard flow, WriteOffExpiredBatchUseCase,
// and a procurement booking's damaged/lost scan replay) automatically
// feeds the same tenant-wide discarded/damaged/lost report -- see
// ListDiscardsUseCase.
func (uc *MarkDamagedOrLostUseCase) Execute(ctx context.Context, in MarkDamagedOrLostInput) (*invDomain.StockMovement, *invDomain.DiscardRecord, error) {
	if in.Type != invDomain.MovementTypeDamaged && in.Type != invDomain.MovementTypeLost {
		return nil, nil, fmt.Errorf("markDamagedOrLost: %w", invDomain.ErrInvalidMovementType)
	}
	if in.Quantity <= 0 {
		return nil, nil, fmt.Errorf("markDamagedOrLost: %w", invDomain.ErrInvalidQuantity)
	}

	var movement invDomain.StockMovement
	var record invDomain.DiscardRecord
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, in.SKUID)
		if err != nil {
			return err
		}
		if err := requireSufficientStock(movements, in.WarehouseID, in.BatchID, -in.Quantity); err != nil {
			return err
		}

		now := time.Now()
		movement = invDomain.StockMovement{
			TenantID:      tenant.IDFromContext(ctx),
			SKUID:         in.SKUID,
			WarehouseID:   in.WarehouseID,
			BatchID:       in.BatchID,
			Type:          in.Type,
			QuantityDelta: -in.Quantity,
			Reason:        in.Reason,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     now,
		}
		if err := uc.movementRepo.Create(ctx, &movement); err != nil {
			return err
		}

		record = invDomain.DiscardRecord{
			TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID, WarehouseID: in.WarehouseID, BatchID: in.BatchID,
			MovementID: movement.ID, Type: in.Type, Quantity: in.Quantity, Reason: in.Reason, PhotoURL: in.PhotoURL,
			DiscardedBy: in.DiscardedBy, DiscardedAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.discardRepo.Create(ctx, &record); err != nil {
			return err
		}

		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("markDamagedOrLost: %w", err)
	}
	return &movement, &record, nil
}
