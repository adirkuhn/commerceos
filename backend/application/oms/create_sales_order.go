package oms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
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
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo        skuLookupRepo
	recordMovement recordSaleMovementUC
}

func NewCreateSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC) *CreateSalesOrderUseCase {
	return &CreateSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, skuRepo: skuRepo, recordMovement: recordMovement}
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
		newOrder := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: source,
			ExternalOrderID: uuid.NewString(), Number: in.Number, OrderKey: in.OrderKey, Status: status,
			WarehouseID: in.WarehouseID, CustomerID: in.CustomerID, SalesChannelID: in.SalesChannelID, Billing: in.Billing, Shipping: in.Shipping,
			CustomerNote: in.CustomerNote, CurrencyCode: in.CurrencyCode,
			DiscountTotalMinorUnits: in.DiscountTotalMinorUnits, ShippingTotalMinorUnits: in.ShippingTotalMinorUnits,
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
		for _, li := range in.LineItems {
			// Best-effort only: unlike the POS/Woo paths, this use case has
			// never validated SKUID against the skus table (the FK
			// constraint is the only thing that ever has) -- a lookup
			// failure here must not turn into a new way for order creation
			// to fail. TaxClassID just stays nil (see its doc comment).
			var taxClassID *string
			if sku, err := uc.skuRepo.FindByID(ctx, li.SKUID); err == nil {
				taxClassID = sku.TaxClassID
			}
			skuID := li.SKUID
			lineItem := omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: newOrder.ID, SKUID: &skuID,
				Name: li.Name, SKUCode: li.SKUCode, Quantity: li.Quantity,
				PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: li.SubtotalMinorUnits,
				SubtotalTaxMinorUnits: li.SubtotalTaxMinorUnits, TotalMinorUnits: li.TotalMinorUnits,
				TotalTaxMinorUnits: li.TotalTaxMinorUnits, TaxClassID: taxClassID, CreatedAt: now,
			}
			if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
				return err
			}
			createdLineItems = append(createdLineItems, lineItem)
		}
		order = &newOrder
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("createSalesOrder: %w", err)
	}
	result := &CreateSalesOrderResult{SalesOrder: *order, LineItems: createdLineItems}
	if !stockReducing {
		return result, nil
	}

	for _, li := range createdLineItems {
		if li.SKUID == nil {
			continue
		}
		movements, err := uc.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: in.WarehouseID, Type: invDomain.MovementTypeSale,
			Quantity: li.Quantity, Reason: fmt.Sprintf("manual sale %s", order.ID),
			OccurredAt: in.OccurredAt,
		})
		if err != nil {
			return nil, fmt.Errorf("createSalesOrder: %w", err)
		}
		for _, movement := range movements {
			if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement); err != nil {
				return nil, fmt.Errorf("createSalesOrder: %w", err)
			}
		}
	}
	return result, nil
}
