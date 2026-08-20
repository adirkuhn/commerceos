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

type ReceiveStockInput struct {
	SKUID       string
	WarehouseID string
	Quantity    int64 // always positive
	// BatchID targets an existing batch. If nil, a new batch is created
	// from LotNumber/ExpiryDate in the same transaction.
	BatchID    *string
	LotNumber  string
	ExpiryDate *time.Time
	Reason     string
	OccurredAt time.Time
	// PurchaseOrderLineItemID and UnitCostMinorUnits are both nil unless
	// this receipt is against a specific procurement.PurchaseOrderLineItem
	// -- the caller (application/procurement, or the desktop receiving
	// review flow via the landed-cost endpoint) is responsible for
	// resolving UnitCostMinorUnits (the line's own cost plus its share of
	// the PO's shipping/fees) before calling Execute; this use case only
	// persists whatever it's given onto the new batch, it has no
	// knowledge of procurement's allocation math (keeps this domain free
	// of a cross-domain repository dependency -- see CLAUDE.md). Both are
	// ignored when BatchID is set (an existing batch's cost was already
	// fixed when it was created).
	PurchaseOrderLineItemID *string
	UnitCostMinorUnits      *int64
}

type ReceiveStockResult struct {
	Batch    *invDomain.Batch // nil when BatchID was provided (no new batch created)
	Movement *invDomain.StockMovement
}

// ReceiveStockUseCase is the atomic "book received stock" action used by
// the Phase 4 receiving flow (see application/procurement's
// ApplyReceivingLineUseCase) and, later, the Phase 5 scanner app -- same
// atomic-composition shape as TransferStockUseCase: optionally create a
// batch, then record a receipt movement, both in one transaction so the
// ledger can never show a movement referencing a batch that didn't
// actually get created.
type ReceiveStockUseCase struct {
	db           *gorm.DB
	batchRepo    invDomain.BatchRepository
	movementRepo invDomain.StockMovementRepository
	outboxRepo   outbox.Repository
}

func NewReceiveStockUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, movementRepo invDomain.StockMovementRepository, outboxRepo outbox.Repository) *ReceiveStockUseCase {
	return &ReceiveStockUseCase{db: db, batchRepo: batchRepo, movementRepo: movementRepo, outboxRepo: outboxRepo}
}

func (uc *ReceiveStockUseCase) Execute(ctx context.Context, in ReceiveStockInput) (*ReceiveStockResult, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("receiveStock: %w", invDomain.ErrInvalidQuantity)
	}

	var result ReceiveStockResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		tenantID := tenant.IDFromContext(ctx)
		now := time.Now()

		batchID := in.BatchID
		if batchID == nil {
			receivedAt := in.OccurredAt
			if receivedAt.IsZero() {
				receivedAt = now
			}
			batch := invDomain.Batch{
				TenantID: tenantID, SKUID: in.SKUID, WarehouseID: in.WarehouseID,
				LotNumber: in.LotNumber, ExpiryDate: in.ExpiryDate, ReceivedAt: receivedAt,
				PurchaseOrderLineItemID: in.PurchaseOrderLineItemID, LandedCostMinorUnits: in.UnitCostMinorUnits,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := uc.batchRepo.Create(ctx, &batch); err != nil {
				return err
			}
			result.Batch = &batch
			batchID = &batch.ID
		}

		movement := invDomain.StockMovement{
			TenantID: tenantID, SKUID: in.SKUID, WarehouseID: in.WarehouseID, BatchID: batchID,
			Type: invDomain.MovementTypeReceipt, QuantityDelta: in.Quantity,
			Reason: in.Reason, OccurredAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.movementRepo.Create(ctx, &movement); err != nil {
			return err
		}
		result.Movement = &movement

		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("receiveStock: %w", err)
	}
	return &result, nil
}
