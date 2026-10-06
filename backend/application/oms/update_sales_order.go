package oms

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/tenant"
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
	// PaymentProvider/PaymentExternalID are stamped by
	// application/payments.WebhookDispatchHandler at the same moment
	// TransactionID is -- see SalesOrder.PaymentProvider's doc.
	PaymentProvider   *string
	PaymentExternalID *string
	DatePaid          *time.Time
	DateCompleted     *time.Time

	// Fulfilled is a manual override -- set explicitly to flip the order's
	// secondary fulfillment status by hand (e.g. a manual/POS order that
	// shipped without ever going through a picking session). When Status
	// is ALSO transitioning to Completed in this same call, an explicit
	// Fulfilled here always wins over the automatic completed-implies-
	// fulfilled behavior below. Normally set by CompletePickingUseCase
	// instead, which always forces it true regardless of pick gaps.
	Fulfilled *bool
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
	db              *gorm.DB
	salesOrderRepo  omsDomain.SalesOrderRepository
	lineItemRepo    omsDomain.SalesOrderLineItemRepository
	linkRepo        omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo         skuLookupRepo
	recordMovement  recordSaleMovementUC
	reverseMovement movementReverser
	outboxRepo      outbox.Repository
}

func NewUpdateSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC, reverseMovement movementReverser, outboxRepo outbox.Repository) *UpdateSalesOrderUseCase {
	return &UpdateSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, skuRepo: skuRepo, recordMovement: recordMovement, reverseMovement: reverseMovement, outboxRepo: outboxRepo}
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
		if in.PaymentProvider != nil {
			fields["payment_provider"] = *in.PaymentProvider
			updated.PaymentProvider = *in.PaymentProvider
		}
		if in.PaymentExternalID != nil {
			fields["payment_external_id"] = *in.PaymentExternalID
			updated.PaymentExternalID = *in.PaymentExternalID
		}
		if in.DatePaid != nil {
			fields["date_paid"] = *in.DatePaid
			updated.DatePaid = in.DatePaid
		}
		if in.DateCompleted != nil {
			fields["date_completed"] = *in.DateCompleted
			updated.DateCompleted = in.DateCompleted
		}

		if in.Fulfilled != nil {
			fields["fulfilled"] = *in.Fulfilled
			updated.Fulfilled = *in.Fulfilled
		}

		statusChanging := in.Status != nil && *in.Status != existing.Status
		if statusChanging {
			fields["status"] = string(*in.Status)
			updated.Status = *in.Status
			if err := enqueueSalesOrderStatusChanged(ctx, uc.outboxRepo, id); err != nil {
				return err
			}
		}
		// Completing an order (any path -- a manual edit here, or
		// ImportWooOrdersUseCase.routeToExisting syncing a Woo status
		// change through this same use case) also marks it fulfilled,
		// same as CompletePickingUseCase always does on finish -- unless
		// the caller already set Fulfilled explicitly in this same call,
		// which always wins.
		if statusChanging && *in.Status == omsDomain.SalesOrderStatusCompleted && in.Fulfilled == nil && !existing.Fulfilled {
			fields["fulfilled"] = true
			updated.Fulfilled = true
		}

		fields["updated_at"] = time.Now()
		if err := uc.salesOrderRepo.Update(ctx, id, fields); err != nil {
			return err
		}

		// StockDecremented flips through its own conditional write, never
		// through the fields map above -- FindByID+Update here is a
		// classic check-then-act with no row lock, so two concurrent
		// Execute calls for the same order (a manual re-fetch racing the
		// scheduler, or two scheduler replicas) could otherwise both read
		// StockDecremented=false and both decrement. TrySetStockDecremented
		// makes the flip itself atomic: only the caller that actually wins
		// it proceeds to the movement loop below -- the other treats it as
		// a no-op, since the winner is already handling it.
		if statusChanging {
			switch {
			case !existing.StockDecremented && in.Status.ReducesStock():
				swapped, err := uc.salesOrderRepo.TrySetStockDecremented(ctx, id, false, true)
				if err != nil {
					return err
				}
				if swapped {
					decrementing = true
					updated.StockDecremented = true
				}
			case existing.StockDecremented && in.Status.Restocks():
				swapped, err := uc.salesOrderRepo.TrySetStockDecremented(ctx, id, true, false)
				if err != nil {
					return err
				}
				if swapped {
					restocking = true
					updated.StockDecremented = false
				}
			}
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

	if decrementing {
		if err := decrementLineItems(ctx, uc.db, uc.lineItemRepo, uc.linkRepo, uc.salesOrderRepo, uc.skuRepo, uc.recordMovement, uc.outboxRepo, id, updated.WarehouseID, lineItems); err != nil {
			return nil, fmt.Errorf("updateSalesOrder: %w", err)
		}
	}
	if restocking {
		if err := restockLineItems(ctx, uc.db, uc.lineItemRepo, uc.salesOrderRepo, uc.reverseMovement, uc.linkRepo, uc.outboxRepo, id, updated.WarehouseID, lineItems); err != nil {
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
