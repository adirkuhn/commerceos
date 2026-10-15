package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ReceivePOSSalesOrderLineItemInput struct {
	// SKUID nil means an adhoc/custom item not tied to a catalog SKU (a
	// POS "open item"/custom-amount line) -- see SalesOrderLineItem's own
	// doc comment. Name is required when nil.
	SKUID    *string
	Name     string
	SKUCode  string
	Quantity int64
	// WeightGrams is set only for a weighed (per-kilo) line -- see
	// SalesOrderLineItem.WeightGrams. Quantity must then be 1.
	WeightGrams *int64

	PriceMinorUnits       int64
	SubtotalMinorUnits    int64
	SubtotalTaxMinorUnits int64
	TotalMinorUnits       int64
	TotalTaxMinorUnits    int64
}

// ReceivePOSSalesOrderInput's authoritative totals (TotalMinorUnits etc.)
// come straight from the POS, the same way Woo treats order.total as
// authoritative rather than re-deriving it from line items -- the POS
// already applied whatever discount/tax/shipping rules it has, CommerceOS
// doesn't re-implement them.
type ReceivePOSSalesOrderInput struct {
	ExternalOrderID string
	Number          string
	OrderKey        string
	Status          omsDomain.SalesOrderStatus // defaults to Completed if empty -- a POS sale already happened
	WarehouseID     string
	CustomerID      *string
	Billing         omsDomain.Address
	Shipping        omsDomain.Address
	CustomerNote    string

	CurrencyCode            string
	DiscountTotalMinorUnits int64
	ShippingTotalMinorUnits int64
	TaxTotalMinorUnits      int64
	TotalMinorUnits         int64

	PaymentMethod      string
	PaymentMethodTitle string
	TransactionID      string
	DatePaid           *time.Time
	DateCompleted      *time.Time

	OccurredAt time.Time
	LineItems  []ReceivePOSSalesOrderLineItemInput
}

// recordSaleMovementUC is *inventory.AllocateSaleStockUseCase, narrowed to
// point of use (ISP) -- picks real batches (FEFO, then FIFO, then the
// general ledger) instead of only ever checking the general ledger, and can
// return more than one movement when a line item's quantity spans several
// batches.
type recordSaleMovementUC interface {
	Execute(ctx context.Context, in invapp.AllocateSaleStockInput) ([]invDomain.StockMovement, error)
}

// movementReverser is *inventory.RecordStockMovementUseCase, narrowed to
// point of use (ISP) -- used only to write a restock's per-batch reversal
// movements (Type Adjustment, positive delta, one call per batch a prior
// decrement actually hit -- see restockLineItems/DeleteSalesOrderUseCase).
// AllocateSaleStockUseCase always removes stock and rejects a positive
// delta, so restocking can't go through it.
type movementReverser interface {
	Execute(ctx context.Context, in invapp.RecordStockMovementInput) (*invDomain.StockMovement, error)
}

// skuLookupRepo is pmsDomain.SKURepository narrowed to point of use (ISP).
type skuLookupRepo interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
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
// tenant.WithTenant must never nest). Movements are recorded with
// ReconcileShortfall so the sale itself is never rejected for insufficient
// tracked stock -- a POS sale already physically happened, so the ledger
// gets a reconciling adjustment instead of blocking the movement (this was
// the concrete cause of orders landing with zero stock movement: the SKU
// had no prior receipt in this warehouse at all). What's still a gap,
// same composition shape and accepted worst-case tradeoff as
// procurement.CompleteBookingUseCase's replay: a crash (DB/network, not
// insufficient stock) between the order being created and every movement
// applying leaves the order recorded with some/all of its stock impact
// still pending, and a retry's idempotency check would then skip
// re-applying the remaining movements too (it only checks "does the order
// exist", not "did every line's movement land"). Accepted for now, same
// category of gap as elsewhere in this codebase -- revisit if POS retries
// turn out to hit this often in practice.
type ReceivePOSSalesOrderUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo        skuLookupRepo
	recordMovement recordSaleMovementUC
}

func NewReceivePOSSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC, outbox eventrecord.Enqueuer) *ReceivePOSSalesOrderUseCase {
	return &ReceivePOSSalesOrderUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, skuRepo: skuRepo, recordMovement: recordMovement}
}

func (uc *ReceivePOSSalesOrderUseCase) Execute(ctx context.Context, in ReceivePOSSalesOrderInput) (*omsDomain.SalesOrder, error) {
	if len(in.LineItems) == 0 {
		return nil, fmt.Errorf("receivePOSSalesOrder: %w", omsDomain.ErrEmptyLineItems)
	}
	if err := requireWeighedLinesAreOnePackage(in.LineItems); err != nil {
		return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
	}
	status := in.Status
	if status == "" {
		status = omsDomain.SalesOrderStatusCompleted
	}

	var order *omsDomain.SalesOrder
	var alreadyReceived bool
	var soldLines []soldLine
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
		newOrder := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: in.ExternalOrderID, Number: in.Number, OrderKey: in.OrderKey, Status: status,
			WarehouseID: in.WarehouseID, CustomerID: in.CustomerID, Billing: in.Billing, Shipping: in.Shipping,
			CustomerNote: in.CustomerNote, CurrencyCode: in.CurrencyCode,
			DiscountTotalMinorUnits: in.DiscountTotalMinorUnits, ShippingTotalMinorUnits: in.ShippingTotalMinorUnits,
			TaxTotalMinorUnits: in.TaxTotalMinorUnits, TotalMinorUnits: in.TotalMinorUnits,
			PaymentMethod: in.PaymentMethod, PaymentMethodTitle: in.PaymentMethodTitle, TransactionID: in.TransactionID,
			DatePaid: in.DatePaid, DateCompleted: in.DateCompleted,
			OccurredAt: in.OccurredAt, CreatedAt: now, UpdatedAt: now,
			// See SalesOrderStatus.ReducesStock's doc -- a POS sale defaults
			// to completed (always stock-reducing), but the status is now
			// exposed to the input so an already-refunded/cancelled sale
			// synced in from elsewhere doesn't get decremented.
			StockDecremented: status.ReducesStock(),
			// A POS sale defaults to Completed (see status above), which
			// means it also defaults to fulfilled -- see
			// SalesOrder.Fulfilled's doc.
			Fulfilled: status == omsDomain.SalesOrderStatusCompleted,
		}
		if err := uc.salesOrderRepo.Create(ctx, &newOrder); err != nil {
			return err
		}
		for _, li := range in.LineItems {
			var lineItem omsDomain.SalesOrderLineItem
			var sku *pmsDomain.SKU
			if li.SKUID == nil || *li.SKUID == "" {
				// Adhoc item -- no catalog SKU to look up, so Name is the
				// only identifying data this line carries at all.
				if li.Name == "" {
					return omsDomain.ErrAdhocLineItemRequiresName
				}
				lineItem = omsDomain.SalesOrderLineItem{
					TenantID: tenant.IDFromContext(ctx), SalesOrderID: newOrder.ID, SKUID: nil,
					Name: li.Name, SKUCode: li.SKUCode, Quantity: li.Quantity, WeightGrams: li.WeightGrams,
					PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: li.SubtotalMinorUnits,
					SubtotalTaxMinorUnits: li.SubtotalTaxMinorUnits, TotalMinorUnits: li.TotalMinorUnits,
					TotalTaxMinorUnits: li.TotalTaxMinorUnits, CreatedAt: now,
				}
			} else {
				sku, err = uc.skuRepo.FindByID(ctx, *li.SKUID)
				if err != nil {
					return fmt.Errorf("receivePOSSalesOrder: line item sku %s: %w", *li.SKUID, err)
				}
				skuID := *li.SKUID
				lineItem = omsDomain.SalesOrderLineItem{
					TenantID: tenant.IDFromContext(ctx), SalesOrderID: newOrder.ID, SKUID: &skuID,
					Name: li.Name, SKUCode: sku.SKUCode, Barcode: sku.Barcode, Quantity: li.Quantity, WeightGrams: li.WeightGrams,
					PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: li.SubtotalMinorUnits,
					SubtotalTaxMinorUnits: li.SubtotalTaxMinorUnits, TotalMinorUnits: li.TotalMinorUnits,
					TotalTaxMinorUnits: li.TotalTaxMinorUnits, TaxClassID: sku.TaxClassID, CreatedAt: now,
				}
			}
			if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
				return err
			}
			soldLines = append(soldLines, newSoldLine(lineItem, sku))
		}
		order = &newOrder
		return recordSalesOrderCreated(ctx, uc.outbox, newOrder, len(soldLines))
	})
	if err != nil {
		return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
	}
	if alreadyReceived || !status.ReducesStock() {
		return order, nil
	}

	for _, sold := range soldLines {
		li := sold.lineItem
		if li.SKUID == nil {
			continue
		}
		movements, err := uc.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: in.WarehouseID, Type: invDomain.MovementTypeSale,
			Quantity: sold.stockQuantity, Reason: fmt.Sprintf("pos sale %s", in.ExternalOrderID),
			OccurredAt: in.OccurredAt,
			// The POS sale already physically happened -- an untracked
			// receipt in CommerceOS must not block recording it. See
			// AllocateSaleStockInput.ReconcileShortfall.
			ReconcileShortfall: true,
		})
		if err != nil {
			return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
		}
		for _, movement := range movements {
			if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement); err != nil {
				return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
			}
		}
		if err := freezeCostAtSale(ctx, uc.db, uc.lineItemRepo, uc.skuRepo, li.ID, *li.SKUID, sold.stockQuantity); err != nil {
			return nil, fmt.Errorf("receivePOSSalesOrder: %w", err)
		}
	}
	return order, nil
}

// soldLine pairs a created line item with how much of its SKU's stock the
// sale consumes -- known only while the SKU is at hand.
type soldLine struct {
	lineItem      omsDomain.SalesOrderLineItem
	stockQuantity int64
}

// newSoldLine takes a weight pool's stock (counted in grams) by the line's
// weight, and every other SKU's by its quantity. sku is nil for an adhoc
// line, which has no stock to take.
func newSoldLine(lineItem omsDomain.SalesOrderLineItem, sku *pmsDomain.SKU) soldLine {
	isWeightPoolSale := sku != nil && sku.UnitType == pmsDomain.SKUUnitTypeKilo && lineItem.WeightGrams != nil
	if isWeightPoolSale {
		return soldLine{lineItem: lineItem, stockQuantity: *lineItem.WeightGrams}
	}
	return soldLine{lineItem: lineItem, stockQuantity: lineItem.Quantity}
}

func requireWeighedLinesAreOnePackage(lineItems []ReceivePOSSalesOrderLineItemInput) error {
	for _, li := range lineItems {
		if li.WeightGrams != nil && li.Quantity != 1 {
			return omsDomain.ErrWeighedLineItemQuantityMustBeOne
		}
	}
	return nil
}
