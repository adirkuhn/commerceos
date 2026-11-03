package channels

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"time"

	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (uc *ImportWooOrdersUseCase) createOne(ctx context.Context, in ImportWooOrdersInput, wo integrationsDomain.WooOrder, externalOrderID string, status omsDomain.SalesOrderStatus, customerID *string, result *ImportOrdersResult) {
	occurredAt, err := parseWooGMTTime(wo.DateCreated)
	if err != nil {
		occurredAt = time.Now()
	}
	var datePaid, dateCompleted *time.Time
	if t, err := parseWooGMTTime(wo.DatePaid); err == nil {
		datePaid = &t
	}
	if t, err := parseWooGMTTime(wo.DateCompleted); err == nil {
		dateCompleted = &t
	}

	totalMinorUnits, _ := ParseWooPriceToMinorUnits(wo.Total)
	discountMinorUnits, _ := ParseWooPriceToMinorUnits(wo.DiscountTotal)
	shippingMinorUnits, _ := ParseWooPriceToMinorUnits(wo.ShippingTotal)
	taxMinorUnits, _ := ParseWooPriceToMinorUnits(wo.TotalTax)
	stockReducing := status.ReducesStock() && !in.SkipStockMovements

	var order omsDomain.SalesOrder
	var createdLineItems []omsDomain.SalesOrderLineItem
	skippedLines := 0
	unlinkedLines := 0
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		order = omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Number: wo.Number,
			Status: status, WarehouseID: in.DefaultWarehouseID, CustomerID: customerID,
			Billing: wooAddressToOMS(wo.Billing), Shipping: wooAddressToOMS(wo.Shipping),
			CustomerNote: wo.CustomerNote, CurrencyCode: wo.Currency,
			DiscountTotalMinorUnits: discountMinorUnits, ShippingTotalMinorUnits: shippingMinorUnits,
			TaxTotalMinorUnits: taxMinorUnits, TotalMinorUnits: totalMinorUnits,
			PaymentMethod: wo.PaymentMethod, PaymentMethodTitle: wo.PaymentMethodTitle,
			ShippingMethod: wo.ShippingMethod, ShippingMethodTitle: wo.ShippingMethodTitle, TransactionID: wo.TransactionID,
			DatePaid: datePaid, DateCompleted: dateCompleted,
			OccurredAt: occurredAt, CreatedAt: now, UpdatedAt: now,
			StockDecremented: stockReducing,
			// An order arriving already-completed (e.g. imported well
			// after the fact) starts fulfilled too -- see
			// SalesOrder.Fulfilled's doc. "processing" naturally starts
			// unfulfilled with no special-casing needed.
			Fulfilled: status == omsDomain.SalesOrderStatusCompleted,
		}
		if err := uc.salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		origin := integrationsDomain.SalesOrderOriginInput{
			TenantID: order.TenantID, SalesOrderID: order.ID, Origin: integrationsDomain.OriginWooCommerce,
			ConnectionID: &in.ConnectionID, ExternalOrderID: externalOrderID, ExternalKey: wo.OrderKey, CreatedAt: now,
		}
		if err := uc.origins.Record(ctx, origin); err != nil {
			return err
		}
		for _, li := range wo.LineItems {
			// A refund/adjustment on the Woo side can surface as a
			// negative (or zero) quantity directly on an order's
			// line_items -- the DB rejects that outright
			// (sales_order_line_items_quantity_check), and there's no
			// stock-moving meaning for it here. Skip it, same tolerance
			// as an unlinked SKU below, rather than letting the INSERT
			// fail and abort every other valid line on the same order.
			if li.Quantity <= 0 {
				skippedLines++
				continue
			}
			sku, err := uc.resolveLineItemSKU(ctx, li.SKU, li.Name)
			if err != nil {
				return err
			}
			var skuID *string
			var taxClassID *string
			if sku != nil {
				skuID = &sku.ID
				taxClassID = sku.TaxClassID
			} else {
				unlinkedLines++
			}
			price, _ := ParseWooPriceToMinorUnits(li.Price)
			subtotal, _ := ParseWooPriceToMinorUnits(li.Subtotal)
			subtotalTax, _ := ParseWooPriceToMinorUnits(li.SubtotalTax)
			total, _ := ParseWooPriceToMinorUnits(li.Total)
			totalTax, _ := ParseWooPriceToMinorUnits(li.TotalTax)
			// A line that owes a decrement (stockReducing) but has no
			// sku_id yet to decrement against starts life Pending -- it
			// can't be attempted at all until LinkLineItemSKUUseCase gives
			// it one. Everything else (matched-SKU lines, or any line on a
			// non-stock-reducing order) starts Decremented, the "no
			// decrement is currently owed" default -- a matched-SKU line
			// only moves to Pending below if its actual decrement attempt
			// hits ErrInsufficientStock.
			stockStatus := omsDomain.StockStatusDecremented
			if stockReducing && skuID == nil {
				stockStatus = omsDomain.StockStatusPending
			}
			lineItem := omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: skuID,
				Name: li.Name, SKUCode: li.SKU, Quantity: li.Quantity, StockStatus: stockStatus,
				PriceMinorUnits: price, SubtotalMinorUnits: subtotal, SubtotalTaxMinorUnits: subtotalTax,
				TotalMinorUnits: total, TotalTaxMinorUnits: totalTax, TaxClassID: taxClassID, CreatedAt: now,
				SoldExpiryDate: wooLineItemSoldExpiryDate(li.MetaData),
			}
			if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
				return err
			}
			createdLineItems = append(createdLineItems, lineItem)
		}
		if unlinkedLines > 0 {
			if err := uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{"needs_sku_review": true}); err != nil {
				return err
			}
			order.NeedsSKUReview = true
		}
		return recordOrderImported(ctx, uc.outbox, order, origin, len(createdLineItems), unlinkedLines)
	})
	if err != nil {
		if isDuplicateImport(err) {
			// Lost a create race: another concurrent import of this same
			// order (a second scheduler replica, or the manual fetch-order
			// endpoint firing at the same moment) won it first. Route
			// through the same existing-order path importOne uses instead
			// of surfacing this as a failure -- the winner already created
			// (and, via UpdateSalesOrderUseCase's own conditional-write
			// guard, will safely decrement) the order.
			existing, findErr := uc.findExisting(ctx, in.ConnectionID, externalOrderID)
			if findErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, findErr))
				return
			}
			if existing != nil {
				uc.routeToExisting(ctx, existing, wo, externalOrderID, status, customerID, result)
				return
			}
		}
		result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
		return
	}
	result.Created++
	if skippedLines > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %d line item(s) skipped (non-positive quantity, likely a refund/adjustment)", externalOrderID, skippedLines))
	}

	if !stockReducing {
		return
	}
	var pendingLineItemIDs []string
	for _, li := range createdLineItems {
		if li.SKUID == nil {
			// Already Pending + order.NeedsSKUReview from creation above --
			// nothing to attempt until it's linked to a SKU.
			continue
		}
		movements, err := uc.allocateStock.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: in.DefaultWarehouseID, Type: invDomain.MovementTypeSale,
			Quantity: li.Quantity, Reason: fmt.Sprintf("woo order %s", externalOrderID), OccurredAt: occurredAt,
		})
		if err != nil {
			if !errors.Is(err, invDomain.ErrInsufficientStock) {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: stock movement: %v", externalOrderID, err))
				continue
			}
			// Genuinely insufficient tracked stock -- not a caller bug and
			// not worth aborting/flagging the whole run as failed for
			// (see cmd/scheduler/dispatch.go's resultErr). Surfaced instead
			// via order.NeedsStockReview on the "Stock Issues" admin page,
			// resolved with RetryStockDecrementUseCase once stock is fixed.
			if err := setChannelsLineItemStockStatus(ctx, uc.db, uc.lineItemRepo, li.ID, omsDomain.StockStatusPending); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
				continue
			}
			pendingLineItemIDs = append(pendingLineItemIDs, li.ID)
			continue
		}
		for _, movement := range movements {
			if err := linkOrderLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: link movement: %v", externalOrderID, err))
			}
		}
	}
	if len(pendingLineItemIDs) > 0 {
		if err := flagImportedOrderStockShortage(ctx, uc.db, uc.salesOrderRepo, uc.outbox, order.ID, pendingLineItemIDs); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
		}
	}
}

// resolveLineItemSKU matches one Woo order line item to a catalog SKU.
// wooSKU (Woo's own line_items[].sku string) is tried first via an exact
// code match -- the normal, reliable path. When that's blank or doesn't
// match anything, falls back to lineItemName (Woo's line_items[].name)
// against the catalog product's own name -- WooCommerce frequently sends a
// blank sku on an order line snapshot even when the underlying product has
// a real one (a common quirk on variations/manually-edited order lines),
// so a line reaching here isn't necessarily for an unknown product. Returns
// nil, nil (not an error) when neither resolves -- same "unlinked, needs
// review" outcome as before this fallback existed, never a guess.
func (uc *ImportWooOrdersUseCase) resolveLineItemSKU(ctx context.Context, wooSKU, lineItemName string) (*pmsDomain.SKU, error) {
	if wooSKU != "" {
		sku, err := uc.skuRepo.FindByCode(ctx, wooSKU)
		if err == nil {
			return sku, nil
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return nil, err
		}
	}
	if lineItemName == "" {
		return nil, nil
	}
	sku, err := uc.skuRepo.FindByExactProductName(ctx, lineItemName)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return sku, nil
}

// setChannelsLineItemStockStatus/setChannelsOrderNeedsStockReview are
// import_woo_orders.go's own copies of application/oms's unexported
// setLineItemStockStatus/setOrderNeedsStockReview helpers -- same shape (own
// tenant.WithTenant call, never nested), duplicated rather than exported
// across the package boundary for two three-line functions, same convention
// linkOrderLineItemMovement already follows for linkLineItemMovement.
func setChannelsLineItemStockStatus(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, lineItemID string, status omsDomain.StockStatus) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return lineItemRepo.Update(ctx, lineItemID, map[string]any{"stock_status": string(status)})
	})
	if err != nil {
		return fmt.Errorf("setChannelsLineItemStockStatus: %w", err)
	}
	return nil
}

// flagImportedOrderStockShortage flags an imported order for stock review
// and records which lines couldn't take their stock.
func flagImportedOrderStockShortage(ctx context.Context, db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, outbox eventrecord.Enqueuer, orderID string, pendingLineItemIDs []string) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := salesOrderRepo.Update(ctx, orderID, map[string]any{"needs_stock_review": true}); err != nil {
			return err
		}
		return eventrecord.Record(ctx, outbox, events.ChannelOrderStockShortageDetected, orderID, eventrecord.Data{
			"pending_line_item_ids": pendingLineItemIDs,
		})
	})
	if err != nil {
		return fmt.Errorf("flagImportedOrderStockShortage: %w", err)
	}
	return nil
}

// wooLineItemSoldExpiryDate looks up the commerceos-woo plugin's
// commerceos_expiry_date order-item meta key (see
// domain/integrations.WooOrderLineItem.MetaData's doc comment) and parses
// it -- "2006-01-02", the exact format toWooMetaData already writes when
// CommerceOS pushes a batch's expiry to Woo, so this round-trips the same
// value byte-for-byte when it comes back on the order. Missing key,
// unparseable value, or blank value (the plugin never even adds the key
// when the product had no expiry meta set) all mean "no value" -- nil, not
// an error, since Woo not knowing an expiry date is the normal case for
// most products.
func wooLineItemSoldExpiryDate(meta []integrationsDomain.WooMetaData) *time.Time {
	for _, m := range meta {
		if m.Key != "commerceos_expiry_date" || m.Value == "" {
			continue
		}
		t, err := time.Parse("2006-01-02", m.Value)
		if err != nil {
			return nil
		}
		return &t
	}
	return nil
}

// linkOrderLineItemMovement is import_woo_orders.go's own copy of
// application/oms's unexported linkLineItemMovement helper -- same shape
// (a movement's own tenant.WithTenant call, never nested), duplicated
// rather than exported across a package boundary for one three-line
// function.
func linkOrderLineItemMovement(ctx context.Context, db *gorm.DB, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, lineItemID string, movement invDomain.StockMovement) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return linkRepo.Create(ctx, &omsDomain.SalesOrderLineItemStockMovement{
			TenantID: tenant.IDFromContext(ctx), SalesOrderLineItemID: lineItemID,
			StockMovementID: movement.ID, BatchID: movement.BatchID, QuantityDelta: movement.QuantityDelta,
			CreatedAt: time.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("linkOrderLineItemMovement: %w", err)
	}
	return nil
}
