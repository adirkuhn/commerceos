package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateSalesOrderLineItemInput struct {
	SKUID    string
	Name     string
	SKUCode  string
	Quantity int64

	PriceMinorUnits       int64
	SubtotalMinorUnits    int64
	SubtotalTaxMinorUnits int64
	TotalMinorUnits       int64
	TotalTaxMinorUnits    int64
	// TotalIncludesTax: TotalMinorUnits already holds TotalTaxMinorUnits
	// (see SalesOrderLineItem.TotalIncludesTax).
	TotalIncludesTax bool
}

// CreateSalesOrderInput is what staff fill in from the admin UI for a
// manually-entered sale (e.g. a walk-in the POS wasn't used for). Unlike
// ReceivePOSSalesOrderInput there's no ExternalOrderID from an outside
// system -- one is generated so the row still satisfies the
// (tenant_id, source, external_order_id) uniqueness constraint every
// SalesOrder shares regardless of source.
type CreateSalesOrderInput struct {
	// Source defaults to SalesOrderSourceManual when empty -- the original
	// (and still most common) caller of this use case, the admin UI's
	// manual-order form. application/whatsapp's chat checkout is the one
	// other caller, passing SalesOrderSourceWhatsApp explicitly.
	Source      omsDomain.SalesOrderSource
	Number      string
	OrderKey    string
	Status      omsDomain.SalesOrderStatus // defaults to Pending -- unlike a POS push, a manual order is not necessarily already paid/fulfilled
	WarehouseID string
	CustomerID  *string
	// QuoteID links the order to the quote it was created from.
	QuoteID *string
	// SalesChannelID is set only by application/storefront's checkout
	// (SalesOrderSourceInternal) -- see SalesOrder.SalesChannelID's doc
	// comment. Nil for every other caller of this use case.
	SalesChannelID *string
	Billing        omsDomain.Address
	Shipping       omsDomain.Address
	CustomerNote   string

	CurrencyCode            string
	DiscountTotalMinorUnits int64
	ShippingTotalMinorUnits int64
	FeeLabel                string
	FeeTotalMinorUnits      int64
	TaxTotalMinorUnits      int64
	TotalMinorUnits         int64

	PaymentMethod       string
	PaymentMethodTitle  string
	ShippingMethod      string
	ShippingMethodTitle string
	CouponCode          string
	TransactionID       string
	// PaymentProvider/PaymentExternalID are set directly at creation time
	// only by application/storefront.CheckoutCartUseCase, when checkout
	// already verified a card payment succeeded (Stripe Elements) before
	// creating the order -- see SalesOrder.PaymentProvider's own doc
	// comment. Every other caller leaves these empty; the webhook path
	// (application/payments.WebhookDispatchHandler) stamps them later via
	// UpdateSalesOrderUseCase instead, for the hosted-checkout-link flow.
	PaymentProvider   string
	PaymentExternalID string
	DatePaid          *time.Time
	DateCompleted     *time.Time

	OccurredAt time.Time
	LineItems  []CreateSalesOrderLineItemInput
}

type CreateSalesOrderResult struct {
	SalesOrder omsDomain.SalesOrder
	LineItems  []omsDomain.SalesOrderLineItem
}

// CreateSalesOrderUseCase is the admin-UI counterpart to
// ReceivePOSSalesOrderUseCase -- same shape (order+line-items in one
// transaction, then a separate stock movement per line, same accepted
// non-atomicity tradeoff documented on ReceivePOSSalesOrderUseCase), but
// gated on SalesOrderStatus.ReducesStock() rather than unconditionally,
// since a manually-created order might be Pending (no stock impact yet).
type CreateSalesOrderUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	origins        originRecorder
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo        skuLookupRepo
	recordMovement recordSaleMovementUC
	numbers        omsDomain.OrderNumbers
}

// WithNumbers gives an order sent without a number the tenant's next one
// (CO-1001, ...), in the order's own transaction.
func (uc *CreateSalesOrderUseCase) WithNumbers(numbers omsDomain.OrderNumbers) *CreateSalesOrderUseCase {
	uc.numbers = numbers
	return uc
}

func NewCreateSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, origins originRecorder, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC, outbox eventrecord.Enqueuer) *CreateSalesOrderUseCase {
	return &CreateSalesOrderUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, origins: origins, lineItemRepo: lineItemRepo, linkRepo: linkRepo, skuRepo: skuRepo, recordMovement: recordMovement}
}

func (uc *CreateSalesOrderUseCase) Execute(ctx context.Context, in CreateSalesOrderInput) (*CreateSalesOrderResult, error) {
	if len(in.LineItems) == 0 {
		return nil, fmt.Errorf("createSalesOrder: %w", omsDomain.ErrEmptyLineItems)
	}
	status := in.Status
	if status == "" {
		status = omsDomain.SalesOrderStatusPending
	}
	source := in.Source
	if source == "" {
		source = omsDomain.SalesOrderSourceManual
	}
	stockReducing := status.ReducesStock()

	var order *omsDomain.SalesOrder
	var createdLineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		number := in.Number
		if number == "" && uc.numbers != nil {
			var err error
			if number, err = uc.numbers.Next(ctx); err != nil {
				return err
			}
		}
		newOrder := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: source,
			ExternalOrderID: uuid.NewString(), Number: number, OrderKey: in.OrderKey, Status: status,
			WarehouseID: in.WarehouseID, CustomerID: in.CustomerID, QuoteID: in.QuoteID, SalesChannelID: in.SalesChannelID, Billing: in.Billing, Shipping: in.Shipping,
			CustomerNote: in.CustomerNote, CurrencyCode: in.CurrencyCode,
			DiscountTotalMinorUnits: in.DiscountTotalMinorUnits, ShippingTotalMinorUnits: in.ShippingTotalMinorUnits,
			FeeLabel: in.FeeLabel, FeeTotalMinorUnits: in.FeeTotalMinorUnits,
			TaxTotalMinorUnits: in.TaxTotalMinorUnits, TotalMinorUnits: in.TotalMinorUnits,
			PaymentMethod: in.PaymentMethod, PaymentMethodTitle: in.PaymentMethodTitle,
			ShippingMethod: in.ShippingMethod, ShippingMethodTitle: in.ShippingMethodTitle, CouponCode: in.CouponCode, TransactionID: in.TransactionID,
			PaymentProvider: in.PaymentProvider, PaymentExternalID: in.PaymentExternalID,
			DatePaid: in.DatePaid, DateCompleted: in.DateCompleted,
			OccurredAt: in.OccurredAt, CreatedAt: now, UpdatedAt: now,
			StockDecremented: stockReducing,
			// See SalesOrder.Fulfilled's doc -- a manually-created order
			// defaults to Pending (not fulfilled) unless staff explicitly
			// enters it as already Completed.
			Fulfilled: status == omsDomain.SalesOrderStatusCompleted,
		}
		if err := uc.salesOrderRepo.Create(ctx, &newOrder); err != nil {
			return err
		}
		// Made here: the origin carries no external reference.
		if err := uc.origins.Record(ctx, integrationsDomain.SalesOrderOriginInput{
			TenantID: newOrder.TenantID, SalesOrderID: newOrder.ID, Origin: integrationsDomain.Origin(source), CreatedAt: now,
		}); err != nil {
			return err
		}
		for _, li := range in.LineItems {
			lineItem := newOrderLineItem(ctx, uc.skuRepo, newOrder.ID, li, now)
			if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
				return err
			}
			createdLineItems = append(createdLineItems, lineItem)
		}
		order = &newOrder
		return recordSalesOrderCreated(ctx, uc.outbox, newOrder, len(createdLineItems))
	})
	if err != nil {
		return nil, fmt.Errorf("createSalesOrder: %w", err)
	}
	result := &CreateSalesOrderResult{SalesOrder: *order, LineItems: createdLineItems}
	if !stockReducing {
		return result, nil
	}

	// A short line doesn't fail the order (it exists already): it's left
	// pending and the order flagged for Stock issues, like a status change
	// into a stock-reducing status (decrementLineItems).
	pending, err := allocateLineItems(ctx, uc.db, uc.lineItemRepo, uc.linkRepo, uc.skuRepo, uc.recordMovement, uc.outbox,
		in.WarehouseID, createdLineItems, fmt.Sprintf("manual sale %s", order.ID), in.OccurredAt)
	if err != nil {
		return nil, fmt.Errorf("createSalesOrder: %w", err)
	}
	if len(pending) > 0 {
		if err := setOrderNeedsStockReview(ctx, uc.db, uc.salesOrderRepo, uc.outbox, order.ID, true, OrderChangeReasonStockShortfall); err != nil {
			return nil, fmt.Errorf("createSalesOrder: %w", err)
		}
		result.SalesOrder.NeedsStockReview = true
		for _, i := range pending {
			if result.LineItems[i].SKUID != nil {
				result.LineItems[i].StockStatus = omsDomain.StockStatusPending
			}
		}
	}
	return result, nil
}

// newOrderLineItem is a line as the admin form sent it, with the SKU's tax
// class snapshotted. Best-effort only: unlike the POS/Woo paths, this has
// never validated SKUID against the skus table (the FK constraint is the
// only thing that ever has) -- a lookup failure here must not turn into a
// new way for an order write to fail. TaxClassID just stays nil (see its
// doc comment).
func newOrderLineItem(ctx context.Context, skuRepo skuLookupRepo, orderID string, li CreateSalesOrderLineItemInput, now time.Time) omsDomain.SalesOrderLineItem {
	var taxClassID *string
	if sku, err := skuRepo.FindByID(ctx, li.SKUID); err == nil {
		taxClassID = sku.TaxClassID
	}
	skuID := li.SKUID
	return omsDomain.SalesOrderLineItem{
		TenantID: tenant.IDFromContext(ctx), SalesOrderID: orderID, SKUID: &skuID,
		Name: li.Name, SKUCode: li.SKUCode, Quantity: li.Quantity,
		PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: li.SubtotalMinorUnits,
		SubtotalTaxMinorUnits: li.SubtotalTaxMinorUnits, TotalMinorUnits: li.TotalMinorUnits,
		TotalTaxMinorUnits: li.TotalTaxMinorUnits, TotalIncludesTax: li.TotalIncludesTax, TaxClassID: taxClassID, CreatedAt: now,
	}
}
