package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type TransferStockInput struct {
	SKUID           string
	Quantity        int64 // always positive — the amount moving
	FromWarehouseID string
	FromBatchID     *string
	ToWarehouseID   string
	ToBatchID       *string
	Reason          string
	OccurredAt      time.Time
}

type TransferStockResult struct {
	Out *invDomain.StockMovement
	In  *invDomain.StockMovement
}

type TransferStockUseCase struct {
	db           *gorm.DB
	movementRepo invDomain.StockMovementRepository
}

func NewTransferStockUseCase(db *gorm.DB, movementRepo invDomain.StockMovementRepository) *TransferStockUseCase {
	return &TransferStockUseCase{db: db, movementRepo: movementRepo}
}

// Execute records a warehouse/batch transfer as two ledger entries — a
// transfer_out at the source and a transfer_in at the destination — created
// atomically in the same transaction so the ledger can never show one side
// without the other.
func (uc *TransferStockUseCase) Execute(ctx context.Context, in TransferStockInput) (*TransferStockResult, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("transferStock: %w", invDomain.ErrInvalidQuantity)
	}

	var result TransferStockResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		tenantID := tenant.IDFromContext(ctx)
		now := time.Now()

		out := invDomain.StockMovement{
			TenantID:      tenantID,
			SKUID:         in.SKUID,
			WarehouseID:   in.FromWarehouseID,
			BatchID:       in.FromBatchID,
			Type:          invDomain.MovementTypeTransferOut,
			QuantityDelta: -in.Quantity,
			Reason:        in.Reason,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     now,
		}
		if err := uc.movementRepo.Create(ctx, &out); err != nil {
			return err
		}

		inbound := invDomain.StockMovement{
			TenantID:      tenantID,
			SKUID:         in.SKUID,
			WarehouseID:   in.ToWarehouseID,
			BatchID:       in.ToBatchID,
			Type:          invDomain.MovementTypeTransferIn,
			QuantityDelta: in.Quantity,
			Reason:        in.Reason,
			OccurredAt:    in.OccurredAt,
			CreatedAt:     now,
		}
		if err := uc.movementRepo.Create(ctx, &inbound); err != nil {
			return err
		}

		result = TransferStockResult{Out: &out, In: &inbound}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("transferStock: %w", err)
	}
	return &result, nil
}
