package inventory

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	// SuppressEvent skips enqueueing inventory.stock.changed for this
	// movement while still recording it. For reconciliation writes whose
	// whole purpose is "make CommerceOS match what the source already
	// reports" (Woo product/stock import) — echoing the event back would
	// just re-push the same number to the store it came from.
	SuppressEvent bool
	// ReconcileShortfall allows a stock-out movement (negative delta) to
	// proceed even when the tracked ledger doesn't have enough: an
	// adjustment movement for the shortfall is written first (recording
	// that untracked stock existed), then the requested movement, both in
	// the same transaction. For a source where the removal already
	// happened in the real world (e.g. a completed POS sale resolved
	// straight to a SKU -- see pms.ResolveBarcodeUseCase) and the ledger
	// just never saw a matching receipt, rejecting the sale would only
	// make CommerceOS's stock count diverge further from what's actually
	// on the shelf. Has no effect on a non-negative delta or when the
	// ledger already covers it.
	ReconcileShortfall bool
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
		movements, err := uc.movementRepo.ListBySKU(ctx, in.SKUID)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := requireSufficientStock(movements, in.WarehouseID, in.BatchID, in.QuantityDelta); err != nil {
			if errors.Is(err, invDomain.ErrInsufficientStock) {
				batchStockBreakdown := make(map[string]int64)
				var generalStock int64
				for _, m := range movements {
					if m.WarehouseID != in.WarehouseID {
						continue
					}
					if m.BatchID == nil {
						generalStock += m.QuantityDelta
					} else {
						batchStockBreakdown[*m.BatchID] += m.QuantityDelta
					}
				}
				requestedBatch := "nil (general ledger)"
				if in.BatchID != nil {
					requestedBatch = *in.BatchID
				}
				log.Printf("[DEBUG] Insufficient stock for SKU %s in warehouse %s. Requested BatchID: %s, Delta: %d. Current general ledger stock: %d. Other batches stock: %+v",
					in.SKUID, in.WarehouseID, requestedBatch, in.QuantityDelta, generalStock, batchStockBreakdown)
			}
			if !in.ReconcileShortfall || !errors.Is(err, invDomain.ErrInsufficientStock) {
				return err
			}
			shortfall := -in.QuantityDelta - currentStock(movements, in.WarehouseID, in.BatchID)
			adjustment := invDomain.StockMovement{
				TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID, WarehouseID: in.WarehouseID, BatchID: in.BatchID,
				Type: invDomain.MovementTypeAdjustment, QuantityDelta: shortfall,
				Reason:     fmt.Sprintf("reconciliation: untracked stock found for %s", in.Reason),
				OccurredAt: in.OccurredAt, CreatedAt: now,
			}
			if err := uc.movementRepo.Create(ctx, &adjustment); err != nil {
				return err
			}
		}

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
		if in.SuppressEvent {
			return nil
		}
		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("recordStockMovement: %w", err)
	}
	return &movement, nil
}
