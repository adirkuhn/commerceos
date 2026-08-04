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

type RecordStockMovementInput struct {
	SKUID         string
	WarehouseID   string
	BatchID       *string
	Type          invDomain.MovementType
	QuantityDelta int64
	Reason        string
	OccurredAt    time.Time
}

type RecordStockMovementUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

func NewRecordStockMovementUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository, outboxRepo outbox.Repository) *RecordStockMovementUseCase {
	return &RecordStockMovementUseCase{db: db, movementRepo: movementRepo, outboxRepo: outboxRepo}
}

// validateSign enforces that a movement's sign matches what its type means:
// stock coming in (receipt, transfer_in) must be positive, stock going out
// (sale, transfer_out, damaged, lost — a write-off always removes stock)
// must be negative. adjustment is the one type allowed to go either way,
// since a recount correction can move stock up or down.
func validateSign(movementType invDomain.MovementType, delta int64) error {
	if delta == 0 {
		return invDomain.ErrInvalidQuantity
	}
	switch movementType {
	case invDomain.MovementTypeReceipt, invDomain.MovementTypeTransferIn:
		if delta < 0 {
			return invDomain.ErrInvalidQuantity
		}
	case invDomain.MovementTypeSale, invDomain.MovementTypeTransferOut,
		invDomain.MovementTypeDamaged, invDomain.MovementTypeLost:
		if delta > 0 {
			return invDomain.ErrInvalidQuantity
		}
	}
	return nil
}

func (uc *RecordStockMovementUseCase) Execute(ctx context.Context, in RecordStockMovementInput) (*invDomain.StockMovement, error) {
	if err := validateSign(in.Type, in.QuantityDelta); err != nil {
		return nil, fmt.Errorf("recordStockMovement: %w", err)
	}

	var movement invDomain.StockMovement
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		movement = invDomain.StockMovement{
			TenantID:      tenant.IDFromContext(ctx),
			SKUID:         in.SKUID,
			WarehouseID:   in.WarehouseID,
			BatchID:       in.BatchID,
			Type:          in.Type,
			QuantityDelta: in.QuantityDelta,
			Reason:        in.Reason,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     now,
		}
		if err := uc.movementRepo.Create(ctx, &movement); err != nil {
			return err
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("recordStockMovement: %w", err)
	}
	return &movement, nil
}
