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

// UpdateSalesOrderInput edits header fields only -- line items are
// immutable once a sales order is created (delete + recreate for a
// line-item change). A nil field means "leave unchanged"; CustomerID's
// zero value ("") explicitly clears the link back to a guest sale.
type UpdateSalesOrderInput struct {
	Number       *string
	OrderKey     *string
	Status       *omsDomain.SalesOrderStatus
	CustomerID   *string
	Billing      *omsDomain.Address
	Shipping     *omsDomain.Address
	CustomerNote *string

	PaymentMethod       *string
	PaymentMethodTitle  *string
	ShippingMethod      *string
	ShippingMethodTitle *string
	TransactionID       *string
	DatePaid            *time.Time
	DateCompleted       *time.Time
}

// UpdateSalesOrderUseCase changes a sales order's header fields, most
// importantly Status. A status transition drives stock the same way
// WooCommerce does (see SalesOrderStatus.ReducesStock/.Restocks): moving
// into a stock-reducing status for the first time decrements, moving from
// an already-decremented order into cancelled/failed restocks. Refunded
// deliberately does not auto-restock -- that disposition (returned
// sellable, damaged, lost) is a separate staff decision made through the
// stock correction flow.
//
// Same two-phase composition as CreateSalesOrderUseCase/
// ReceivePOSSalesOrderUseCase: the header update (and the line-item read
// needed to know quantities) is one transaction, the stock movements are
// separate calls after it commits.
type UpdateSalesOrderUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	recordMovement recordSaleMovementUC
}

func NewUpdateSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, recordMovement recordSaleMovementUC) *UpdateSalesOrderUseCase {
	return &UpdateSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, recordMovement: recordMovement}
}

func (uc *UpdateSalesOrderUseCase) Execute(ctx context.Context, id string, in UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	var updated omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	var decrementing, restocking bool

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.salesOrderRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		updated = *existing

		fields := map[string]any{}
		if in.Number != nil {
			fields["number"] = *in.Number
			updated.Number = *in.Number
		}
		if in.OrderKey != nil {
			fields["order_key"] = *in.OrderKey
			updated.OrderKey = *in.OrderKey
		}
		if in.CustomerID != nil {
			if *in.CustomerID == "" {
				fields["customer_id"] = nil
				updated.CustomerID = nil
			} else {
				fields["customer_id"] = *in.CustomerID
				updated.CustomerID = in.CustomerID
			}
		}
		if in.Billing != nil {
			setBillingFields(fields, *in.Billing)
			updated.Billing = *in.Billing
		}
		if in.Shipping != nil {
			setShippingFields(fields, *in.Shipping)
			updated.Shipping = *in.Shipping
		}
		if in.CustomerNote != nil {
			fields["customer_note"] = *in.CustomerNote
			updated.CustomerNote = *in.CustomerNote
		}
		if in.PaymentMethod != nil {
			fields["payment_method"] = *in.PaymentMethod
			updated.PaymentMethod = *in.PaymentMethod
		}
		if in.PaymentMethodTitle != nil {
			fields["payment_method_title"] = *in.PaymentMethodTitle
			updated.PaymentMethodTitle = *in.PaymentMethodTitle
		}
		if in.ShippingMethod != nil {
			fields["shipping_method"] = *in.ShippingMethod
			updated.ShippingMethod = *in.ShippingMethod
		}
		if in.ShippingMethodTitle != nil {
			fields["shipping_method_title"] = *in.ShippingMethodTitle
			updated.ShippingMethodTitle = *in.ShippingMethodTitle
		}
		if in.TransactionID != nil {
			fields["transaction_id"] = *in.TransactionID
			updated.TransactionID = *in.TransactionID
		}
		if in.DatePaid != nil {
			fields["date_paid"] = *in.DatePaid
			updated.DatePaid = in.DatePaid
		}
		if in.DateCompleted != nil {
			fields["date_completed"] = *in.DateCompleted
			updated.DateCompleted = in.DateCompleted
		}

		if in.Status != nil && *in.Status != existing.Status {
			fields["status"] = string(*in.Status)
			updated.Status = *in.Status

			switch {
			case !existing.StockDecremented && in.Status.ReducesStock():
				decrementing = true
				fields["stock_decremented"] = true
				updated.StockDecremented = true
			case existing.StockDecremented && in.Status.Restocks():
				restocking = true
				fields["stock_decremented"] = false
				updated.StockDecremented = false
			}
		}

		fields["updated_at"] = time.Now()
		if err := uc.salesOrderRepo.Update(ctx, id, fields); err != nil {
			return err
		}

		if decrementing || restocking {
			lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateSalesOrder: %w", err)
	}

	for _, li := range lineItems {
		if li.SKUID == nil {
			continue
		}
		delta := li.Quantity
		reason := fmt.Sprintf("sales order %s restocked", id)
		if decrementing {
			delta = -li.Quantity
			reason = fmt.Sprintf("sales order %s status change", id)
		}
		movement, err := uc.recordMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: *li.SKUID, WarehouseID: updated.WarehouseID, Type: invDomain.MovementTypeSale,
			QuantityDelta: delta, Reason: reason, OccurredAt: time.Now(),
		})
		if err != nil {
			return nil, fmt.Errorf("updateSalesOrder: %w", err)
		}
		if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement.ID); err != nil {
			return nil, fmt.Errorf("updateSalesOrder: %w", err)
		}
	}
	return &updated, nil
}

func setBillingFields(fields map[string]any, a omsDomain.Address) {
	fields["billing_first_name"] = a.FirstName
	fields["billing_last_name"] = a.LastName
	fields["billing_company"] = a.Company
	fields["billing_address_1"] = a.Address1
	fields["billing_address_2"] = a.Address2
	fields["billing_city"] = a.City
	fields["billing_state"] = a.State
	fields["billing_postcode"] = a.Postcode
	fields["billing_country"] = a.Country
	fields["billing_email"] = a.Email
	fields["billing_phone"] = a.Phone
}

func setShippingFields(fields map[string]any, a omsDomain.Address) {
	fields["shipping_first_name"] = a.FirstName
	fields["shipping_last_name"] = a.LastName
	fields["shipping_company"] = a.Company
	fields["shipping_address_1"] = a.Address1
	fields["shipping_address_2"] = a.Address2
	fields["shipping_city"] = a.City
	fields["shipping_state"] = a.State
	fields["shipping_postcode"] = a.Postcode
	fields["shipping_country"] = a.Country
	fields["shipping_phone"] = a.Phone
}
