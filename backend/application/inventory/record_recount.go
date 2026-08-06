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

type RecordRecountInput struct {
	SKUID           string
	WarehouseID     string
	BatchID         *string // nil = recount the un-lotted, general stock for this SKU
	CountedQuantity int64
	Reason          string
	OccurredAt      time.Time
}

type RecordRecountUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

func NewRecordRecountUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, outboxRepo outbox.Repository) *RecordRecountUseCase {
	return &RecordRecountUseCase{db: db, movementRepo: movementRepo, outboxRepo: outboxRepo}
}

// Execute compares CountedQuantity against the current computed stock for
// the given scope (a batch, or the un-lotted stock for a SKU) and records a
// single adjustment movement for the difference. Returns a nil movement
// (and no error) when the count already matches — there's nothing to
// correct, and QuantityDelta=0 movements are rejected by the domain anyway.
func (uc *RecordRecountUseCase) Execute(ctx context.Context, in RecordRecountInput) (*invDomain.StockMovement, error) {
	if in.CountedQuantity < 0 {
		return nil, fmt.Errorf("recordRecount: %w", invDomain.ErrInvalidQuantity)
	}

	var movement *invDomain.StockMovement
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var current int64
		if in.BatchID != nil {
			movements, err := uc.movementRepo.ListByBatch(ctx, *in.BatchID)
			if err != nil {
				return err
			}
			for _, m := range movements {
				current += m.QuantityDelta
			}
		} else {
			movements, err := uc.movementRepo.ListBySKU(ctx, in.SKUID)
			if err != nil {
				return err
			}
			for _, m := range movements {
				if m.BatchID == nil {
					current += m.QuantityDelta
				}
			}
		}

		delta := in.CountedQuantity - current
		if delta == 0 {
			return nil
		}

		m := invDomain.StockMovement{
			TenantID:      tenant.IDFromContext(ctx),
			SKUID:         in.SKUID,
			WarehouseID:   in.WarehouseID,
			BatchID:       in.BatchID,
			Type:          invDomain.MovementTypeAdjustment,
			QuantityDelta: delta,
			Reason:        in.Reason,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     time.Now(),
		}
		if err := uc.movementRepo.Create(ctx, &m); err != nil {
			return err
		}
		movement = &m
		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("recordRecount: %w", err)
	}
	return movement, nil
}
