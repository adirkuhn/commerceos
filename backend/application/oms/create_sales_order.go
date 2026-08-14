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
	Number       string
	OrderKey     string
	Status       omsDomain.SalesOrderStatus // defaults to Pending -- unlike a POS push, a manual order is not necessarily already paid/fulfilled
	WarehouseID  string
	CustomerID   *string
	Billing      omsDomain.Address
	Shipping     omsDomain.Address
	CustomerNote string

	CurrencyCode            string
	DiscountTotalMinorUnits int64
	ShippingTotalMinorUnits int64
	TaxTotalMinorUnits      int64
	TotalMinorUnits         int64

	PaymentMethod       string
	PaymentMethodTitle  string
	ShippingMethod      string
	ShippingMethodTitle string
	TransactionID       string
	DatePaid            *time.Time
	DateCompleted       *time.Time

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
	recordMovement recordSaleMovementUC
}

func NewCreateSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, recordMovement recordSaleMovementUC) *CreateSalesOrderUseCase {
	return &CreateSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, recordMovement: recordMovement}
}

func (uc *CreateSalesOrderUseCase) Execute(ctx context.Context, in CreateSalesOrderInput) (*CreateSalesOrderResult, error) {
	if len(in.LineItems) == 0 {
		return nil, fmt.Errorf("createSalesOrder: %w", omsDomain.ErrEmptyLineItems)
	}
	status := in.Status
	if status == "" {
		status = omsDomain.SalesOrderStatusPending
	}
	stockReducing := status.ReducesStock()

	var order *omsDomain.SalesOrder
	var createdLineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		newOrder := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceManual,
			ExternalOrderID: uuid.NewString(), Number: in.Number, OrderKey: in.OrderKey, Status: status,
			WarehouseID: in.WarehouseID, CustomerID: in.CustomerID, Billing: in.Billing, Shipping: in.Shipping,
			CustomerNote: in.CustomerNote, CurrencyCode: in.CurrencyCode,
			DiscountTotalMinorUnits: in.DiscountTotalMinorUnits, ShippingTotalMinorUnits: in.ShippingTotalMinorUnits,
			TaxTotalMinorUnits: in.TaxTotalMinorUnits, TotalMinorUnits: in.TotalMinorUnits,
			PaymentMethod: in.PaymentMethod, PaymentMethodTitle: in.PaymentMethodTitle,
			ShippingMethod: in.ShippingMethod, ShippingMethodTitle: in.ShippingMethodTitle, TransactionID: in.TransactionID,
			DatePaid: in.DatePaid, DateCompleted: in.DateCompleted,
			OccurredAt: in.OccurredAt, CreatedAt: now, UpdatedAt: now,
			StockDecremented: stockReducing,
		}
		if err := uc.salesOrderRepo.Create(ctx, &newOrder); err != nil {
			return err
		}
		for _, li := range in.LineItems {
			lineItem := omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: newOrder.ID, SKUID: li.SKUID,
				Name: li.Name, SKUCode: li.SKUCode, Quantity: li.Quantity,
				PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: li.SubtotalMinorUnits,
				SubtotalTaxMinorUnits: li.SubtotalTaxMinorUnits, TotalMinorUnits: li.TotalMinorUnits,
				TotalTaxMinorUnits: li.TotalTaxMinorUnits, CreatedAt: now,
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
		movement, err := uc.recordMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: li.SKUID, WarehouseID: in.WarehouseID, Type: invDomain.MovementTypeSale,
			QuantityDelta: -li.Quantity, Reason: fmt.Sprintf("manual sale %s", order.ID),
			OccurredAt: in.OccurredAt,
		})
		if err != nil {
			return nil, fmt.Errorf("createSalesOrder: %w", err)
		}
		if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement.ID); err != nil {
			return nil, fmt.Errorf("createSalesOrder: %w", err)
		}
	}
	return result, nil
}
