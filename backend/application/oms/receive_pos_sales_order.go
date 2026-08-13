package oms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ReceivePOSSalesOrderLineItemInput struct {
	SKUID               string
	Quantity            int64
	UnitPriceMinorUnits int64
}

type ReceivePOSSalesOrderInput struct {
	ExternalOrderID string
	WarehouseID     string
	CurrencyCode    string
	OccurredAt      time.Time
	LineItems       []ReceivePOSSalesOrderLineItemInput
}

// recordSaleMovementUC is *inventory.RecordStockMovementUseCase, narrowed
// to point of use (ISP).
type recordSaleMovementUC interface {
	Execute(ctx context.Context, in invapp.RecordStockMovementInput) (*invDomain.StockMovement, error)
}

// ReceivePOSSalesOrderUseCase is what POST /pos/sales-orders calls --
// idempotent on (TenantID, Source, ExternalOrderID): the POS retrying an
// already-received sale (e.g. after a dropped response) is a no-op, not a
// duplicate order or a double stock decrement.
//
// Order creation and its stock-decrement movements are NOT one atomic
// transaction: the order+line-items write is one tenant.WithTenant call,
// then each line's stock movement is its own separate call (composing
// RecordStockMovementUseCase, which opens its own transaction --
// tenant.WithTenant must never nest). Same composition shape and the same
// accepted worst-case tradeoff as procurement.CompleteBookingUseCase's
// replay: a crash between the order being created and every movement
// applying leaves the order recorded with some/all of its stock impact
// still pending, and a retry's idempotency check would then skip
// re-applying the remaining movements too (it only checks "does the order
// exist", not "did every line's movement land"). Accepted for now, same
// category of gap as elsewhere in this codebase -- revisit if POS retries
// turn out to hit this often in practice.
type ReceivePOSSalesOrderUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	recordMovement recordSaleMovementUC
}

func NewReceivePOSSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, recordMovement recordSaleMovementUC) *ReceivePOSSalesOrderUseCase {
	return &ReceivePOSSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, recordMovement: recordMovement}
}

func (uc *ReceivePOSSalesOrderUseCase) Execute(ctx context.Context, in ReceivePOSSalesOrderInput) (*omsDomain.SalesOrder, error) {
	if len(in.LineItems) == 0 {
		return nil, fmt.Errorf("receivePOSSalesOrder: %w", omsDomain.ErrEmptyLineItems)
	}

	var order *omsDomain.SalesOrder
	var alreadyReceived bool
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourcePOS, in.ExternalOrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			order = existing
			alreadyReceived = true
			return nil
		}

		now := time.Now()
		var total int64
		for _, li := range in.LineItems {
			total += li.Quantity * li.UnitPriceMinorUnits
		}
		newOrder := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: in.ExternalOrderID, WarehouseID: in.WarehouseID,
			CurrencyCode: in.CurrencyCode, TotalMinorUnits: total,
			OccurredAt: in.OccurredAt, CreatedAt: now,
		}
		if err := uc.salesOrderRepo.Create(ctx, &newOrder); err != nil {
			return err
		}
		for _, li := range in.LineItems {
			lineItem := omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: newOrder.ID, SKUID: li.SKUID,
				Quantity: li.Quantity, UnitPriceMinorUnits: li.UnitPriceMinorUnits, CreatedAt: now,
			}
			if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
				return err
			}
		}
		order = &newOrder
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
	}
	if alreadyReceived {
		return order, nil
	}

	for _, li := range in.LineItems {
		if _, err := uc.recordMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: li.SKUID, WarehouseID: in.WarehouseID, Type: invDomain.MovementTypeSale,
			QuantityDelta: -li.Quantity, Reason: fmt.Sprintf("pos sale %s", in.ExternalOrderID),
			OccurredAt: in.OccurredAt,
		}); err != nil {
			return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
		}
	}
	return order, nil
}
