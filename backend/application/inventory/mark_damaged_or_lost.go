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
}

type MarkDamagedOrLostUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

func NewMarkDamagedOrLostUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, outboxRepo outbox.Repository) *MarkDamagedOrLostUseCase {
	return &MarkDamagedOrLostUseCase{db: db, movementRepo: movementRepo, outboxRepo: outboxRepo}
}

// Execute records a write-off — stock that's gone (damaged or lost) and
// won't be sold. Type must be MovementTypeDamaged or MovementTypeLost;
// anything else is rejected rather than silently written as some other
// kind of movement.
func (uc *MarkDamagedOrLostUseCase) Execute(ctx context.Context, in MarkDamagedOrLostInput) (*invDomain.StockMovement, error) {
	if in.Type != invDomain.MovementTypeDamaged && in.Type != invDomain.MovementTypeLost {
		return nil, fmt.Errorf("markDamagedOrLost: %w", invDomain.ErrInvalidMovementType)
	}
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("markDamagedOrLost: %w", invDomain.ErrInvalidQuantity)
	}

	var movement invDomain.StockMovement
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		movements, err := uc.movementRepo.ListBySKU(ctx, in.SKUID)
		if err != nil {
			return err
		}
		if err := requireSufficientStock(movements, in.WarehouseID, in.BatchID, -in.Quantity); err != nil {
			return err
		}

		movement = invDomain.StockMovement{
			TenantID:      tenant.IDFromContext(ctx),
			SKUID:         in.SKUID,
			WarehouseID:   in.WarehouseID,
			BatchID:       in.BatchID,
			Type:          in.Type,
			QuantityDelta: -in.Quantity,
			Reason:        in.Reason,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     time.Now(),
		}
		if err := uc.movementRepo.Create(ctx, &movement); err != nil {
			return err
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("markDamagedOrLost: %w", err)
	}
	return &movement, nil
}
