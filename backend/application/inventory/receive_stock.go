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

// SKUCostReceiptInput is what ReceiveStockUseCase hands to its
// skuCostUpdater after a costed receipt commits -- a local mirror of
// application/pms's ApplyReceiptCostInput, not an import of application/pms
// (which already imports application/inventory -- see
// application/pms/recalculate_cost_prices.go's own doc comment on that
// exact cycle), wired via a small adapter in the composition root instead.
type SKUCostReceiptInput struct {
	SKUID                    string
	ReceivedQuantity         int64
	LandedUnitCostMinorUnits int64
	// ReferenceType/ReferenceID is "purchase_order_line_item"/<id> when the
	// receipt has one, else "batch"/<id> -- see Execute.
	ReferenceType string
	ReferenceID   string
	OccurredAt    time.Time
}

// skuCostUpdaterForReceiveStock is the narrow slice of application/pms's
// MWAC receipt hook (ApplyReceiptCostUseCase) this use case needs (ISP,
// point of use) -- kept as a local interface rather than an import, same
// cross-domain-avoidance reasoning as SKUCostReceiptInput's doc comment.
type skuCostUpdaterForReceiveStock interface {
	Execute(ctx context.Context, in SKUCostReceiptInput) error
}

// ReceiveStockUseCase is the atomic "book received stock" action used by
// the Phase 4 receiving flow (see application/procurement's
// ApplyReceivingLineUseCase) and, later, the Phase 5 scanner app -- same
// atomic-composition shape as TransferStockUseCase: optionally create a
// batch, then record a receipt movement, both in one transaction so the
// ledger can never show a movement referencing a batch that didn't
// actually get created.
type ReceiveStockUseCase struct {
	db             *gorm.DB
	batchRepo      invDomain.BatchRepository
	movementRepo   invDomain.StockMovementRepository
	outboxRepo     outbox.Repository
	skuCostUpdater skuCostUpdaterForReceiveStock
}

func NewReceiveStockUseCase(db *gorm.DB, batchRepo invDomain.BatchRepository, movementRepo invDomain.StockMovementRepository, outboxRepo outbox.Repository, skuCostUpdater skuCostUpdaterForReceiveStock) *ReceiveStockUseCase {
	return &ReceiveStockUseCase{db: db, batchRepo: batchRepo, movementRepo: movementRepo, outboxRepo: outboxRepo, skuCostUpdater: skuCostUpdater}
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

		if in.PurchaseOrderLineItemID != nil && in.UnitCostMinorUnits != nil {
			if err := enqueueGoodsReceived(ctx, uc.outboxRepo, in.SKUID, in.WarehouseID, *batchID,
				*in.PurchaseOrderLineItemID, in.Quantity, *in.UnitCostMinorUnits); err != nil {
				return err
			}
		}

		return enqueueStockChanged(ctx, uc.outboxRepo, in.SKUID)
	})
	if err != nil {
		return nil, fmt.Errorf("receiveStock: %w", err)
	}

	if in.UnitCostMinorUnits != nil && uc.skuCostUpdater != nil {
		refType, refID := "batch", *result.Movement.BatchID
		if in.PurchaseOrderLineItemID != nil {
			refType, refID = "purchase_order_line_item", *in.PurchaseOrderLineItemID
		}
		occurredAt := in.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		// Deliberately its own transaction, not nested inside the one
		// above (tenant.WithTenant must never nest -- see
		// application/procurement.CompleteBookingUseCase's doc comment on
		// the same composition constraint). The batch and stock movement
		// are already durably committed at this point; a failure here just
		// means the SKU's moving-average cost falls behind until the
		// reconciliation job flags the drift -- same accepted-gap shape as
		// CompleteBookingUseCase.markReplayed.
		if err := uc.skuCostUpdater.Execute(ctx, SKUCostReceiptInput{
			SKUID: in.SKUID, ReceivedQuantity: in.Quantity, LandedUnitCostMinorUnits: *in.UnitCostMinorUnits,
			ReferenceType: refType, ReferenceID: refID, OccurredAt: occurredAt,
		}); err != nil {
			return &result, fmt.Errorf("receiveStock: cost update: %w", err)
		}
	}

	return &result, nil
}
