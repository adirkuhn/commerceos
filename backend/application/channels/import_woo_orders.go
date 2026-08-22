package channels

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// maxOrderImportPages is higher than maxImportPages (products): an active
// store accumulates orders indefinitely, where the product catalog is
// comparatively bounded. At 50/page this is 250,000 orders per run --
// generous enough that hitting it on a real store's first (ModifiedAfter
// nil) full import should be rare, but Execute still reports it via
// result.Errors rather than silently stopping if it ever does.
const maxOrderImportPages = 5000

// ImportWooOrdersInput drives both the "full import" and "incremental
// check" runs off the same use case -- see ModifiedAfter.
type ImportWooOrdersInput struct {
	ConnectionID string
	// DefaultWarehouseID: Woo has no warehouse concept -- every synced
	// order (and the stock movements it produces) is attributed here,
	// same convention as ImportWooProductsInput/ImportWooStockLevelsInput.
	DefaultWarehouseID string
	// ModifiedAfter nil means a full scan of every order on every page --
	// used for a schedule's first-ever run (nothing synced yet, so
	// there's no cursor to be incremental from) and can be forced again
	// any time as a reconciliation pass. Non-nil filters to orders Woo
	// reports modified since that instant, the cheap steady-state path.
	ModifiedAfter *time.Time
	// SkipStockMovements, when true, still creates every order and its
	// line items exactly as normal but never decrements inventory for
	// them, even for a status that would otherwise reduce stock -- used
	// by the one-time bootstrap that runs orders import right after a
	// connection's very first products import: that import already set
	// each SKU's stock to Woo's current, already-net-of-these-orders
	// quantity, so decrementing again here would double-count.
	SkipStockMovements bool
}

type ImportOrdersResult struct {
	Created   int
	Updated   int
	Unchanged int
	Errors    []string
	// LastModifiedAt is the latest date_modified_gmt seen across every
	// order Woo returned this run (regardless of whether that specific
	// order imported successfully) -- nil if no orders were returned at
	// all. cmd/scheduler uses this, not its own wall-clock time.Now(), to
	// advance the next run's ModifiedAfter cursor: tying the cursor to
	// Woo's own clock instead of the poller's avoids a whole class of
	// missed-order bug from clock skew/replication lag between this
	// process and the Woo server (a wall-clock cursor can march past an
	// order whose write to Woo's DB simply hadn't finished/replicated
	// yet at the moment the wall clock ticked past it).
	LastModifiedAt *time.Time
}

// updateSalesOrderUC is application/oms's UpdateSalesOrderUseCase,
// narrowed to point of use (ISP) -- reused rather than reimplemented so a
// Woo status change decrements/restocks through the exact same
// SalesOrderStatus.ReducesStock/.Restocks logic a staff-made edit would.
type updateSalesOrderUC interface {
	Execute(ctx context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error)
}

// ImportWooOrdersUseCase pulls orders from a WooCommerce store into
// domain/oms.SalesOrder. Idempotent per Woo order via
// SalesOrderRepository.FindByExternalID(SourceWooCommerce, wooOrderID) --
// an order not seen before is created (with its own stock decrement, same
// two-phase composition as CreateSalesOrderUseCase/
// ReceivePOSSalesOrderUseCase: header+lines in one transaction, stock
// movements as separate calls after it commits, never nested); an order
// already known whose status changed is routed through
// UpdateSalesOrderUseCase so the stock transition logic isn't duplicated;
// unchanged orders are a no-op. One order's failure is recorded on the
// result and never aborts the rest of the run -- same tolerance as
// products import.
// saleStockAllocator is *inventory.AllocateSaleStockUseCase, narrowed to
// point of use (ISP) -- picks real batches (FEFO, then FIFO, then the
// general ledger) for a Woo order's decrement instead of only ever checking
// the general ledger, splitting across batches when one alone doesn't cover
// a line item's quantity.
type saleStockAllocator interface {
	Execute(ctx context.Context, in invapp.AllocateSaleStockInput) ([]invDomain.StockMovement, error)
}

type ImportWooOrdersUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
	salesOrderRepo   omsDomain.SalesOrderRepository
	lineItemRepo     omsDomain.SalesOrderLineItemRepository
	linkRepo         omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo          pmsDomain.SKURepository
	customerRepo     customersDomain.CustomerRepository
	allocateStock    saleStockAllocator
	updateOrder      updateSalesOrderUC
}

func NewImportWooOrdersUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	linkRepo omsDomain.SalesOrderLineItemStockMovementRepository,
	skuRepo pmsDomain.SKURepository,
	customerRepo customersDomain.CustomerRepository,
	allocateStock saleStockAllocator,
	updateOrder updateSalesOrderUC,
) *ImportWooOrdersUseCase {
	return &ImportWooOrdersUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret,
		salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo,
		skuRepo: skuRepo, customerRepo: customerRepo, allocateStock: allocateStock, updateOrder: updateOrder,
	}
}

func (uc *ImportWooOrdersUseCase) Execute(ctx context.Context, in ImportWooOrdersInput) (*ImportOrdersResult, error) {
	client, _, err := uc.connect(ctx, in.ConnectionID)
	if err != nil {
		return nil, err
	}

	result := ImportOrdersResult{}
	for page := 1; page <= maxOrderImportPages; page++ {
		orders, err := client.ListOrders(ctx, page, 50, in.ModifiedAfter)
		if err != nil {
			// A transient Woo/network failure mid-run must not turn into a
			// hard error here -- the caller (cmd/scheduler) only advances
			// this schedule's incremental cursor (LastRunAt) on a
			// "success" status, specifically so a run that didn't reach
			// every page gets retried from the same window next time
			// instead of silently skipping whatever was never fetched.
			// Returning the partial result (with everything imported so
			// far still counted) rather than discarding it via a hard
			// error keeps that already-real progress visible in this
			// run's report.
			result.Errors = append(result.Errors, fmt.Sprintf("page %d: %v", page, fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)))
			return &result, nil
		}
		if len(orders) == 0 {
			return &result, nil
		}
		for _, wo := range orders {
			if t, err := parseWooGMTTime(wo.DateModified); err == nil {
				if result.LastModifiedAt == nil || t.After(*result.LastModifiedAt) {
					result.LastModifiedAt = &t
				}
			}
			uc.importOne(ctx, in, wo, &result)
		}
	}
	result.Errors = append(result.Errors, fmt.Sprintf("reached the %d-page import cap -- some orders may not have been imported this run", maxOrderImportPages))
	return &result, nil
}

// connect decrypts the connection's stored secret and builds a WooClient
// for it -- shared by Execute and ExecuteOne so the two never drift.
// Returns the Connection too since ExecuteOne needs its DefaultWarehouseID.
func (uc *ImportWooOrdersUseCase) connect(ctx context.Context, connectionID string) (integrationsDomain.WooClient, *integrationsDomain.Connection, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *existing
		secret, err = crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("importWooOrders: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, nil, fmt.Errorf("importWooOrders: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return client, &conn, nil
}

// FetchWooOrderInput drives ExecuteOne -- the manual "fetch this specific
// known order" admin action (see FetchWooOrderResult).
type FetchWooOrderInput struct {
	ConnectionID       string
	ExternalOrderID    string // the order's own Woo ID/number, as shown in the Woo admin
	DefaultWarehouseID string
}

// FetchWooOrderResult mirrors ImportOrdersResult's Created/Updated/
// Unchanged/Errors shape, narrowed to the one order ExecuteOne fetched.
type FetchWooOrderResult struct {
	ExternalOrderID string
	// SalesOrderID is "" only when Errors is non-empty -- every other
	// outcome (created, updated, or already unchanged) resolves to a real
	// CommerceOS sales order id.
	SalesOrderID string
	Created      bool
	Updated      bool
	Unchanged    bool
	Errors       []string
}

// ExecuteOne fetches exactly one order by its Woo order ID and runs it
// through the same import/update logic Execute's page loop uses (importOne/
// createOne) -- the manual counterpart to the scheduled incremental sync,
// for backfilling one specific known-missing order on demand rather than
// waiting for (or in place of) the next scheduled run.
func (uc *ImportWooOrdersUseCase) ExecuteOne(ctx context.Context, in FetchWooOrderInput) (*FetchWooOrderResult, error) {
	client, conn, err := uc.connect(ctx, in.ConnectionID)
	if err != nil {
		return nil, err
	}

	wo, err := client.GetOrder(ctx, in.ExternalOrderID)
	if err != nil {
		return nil, fmt.Errorf("importWooOrders: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	// DefaultWarehouseID falls back to the connection's own configured
	// default -- unlike a scheduled sync (always driven by a SyncSchedule
	// row that requires one), this manual one-order fetch has no such row
	// to require it from the caller.
	warehouseID := in.DefaultWarehouseID
	if warehouseID == "" && conn.DefaultWarehouseID != nil {
		warehouseID = *conn.DefaultWarehouseID
	}

	importResult := ImportOrdersResult{}
	uc.importOne(ctx, ImportWooOrdersInput{
		ConnectionID: in.ConnectionID, DefaultWarehouseID: warehouseID,
	}, *wo, &importResult)

	result := &FetchWooOrderResult{
		ExternalOrderID: in.ExternalOrderID,
		Created:         importResult.Created > 0,
		Updated:         importResult.Updated > 0,
		Unchanged:       importResult.Unchanged > 0,
		Errors:          importResult.Errors,
	}
	if len(result.Errors) == 0 {
		err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			so, err := uc.salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourceWooCommerce, in.ExternalOrderID)
			if err != nil {
				return err
			}
			if so != nil {
				result.SalesOrderID = so.ID
			}
			return nil
		})
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}
	return result, nil
}

func (uc *ImportWooOrdersUseCase) importOne(ctx context.Context, in ImportWooOrdersInput, wo integrationsDomain.WooOrder, result *ImportOrdersResult) {
	externalOrderID := strconv.Itoa(wo.ID)
	status := omsDomain.SalesOrderStatus(wo.Status)

	var existing *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		existing, err = uc.salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourceWooCommerce, externalOrderID)
		return err
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
		return
	}

	if existing != nil {
		uc.routeToExisting(ctx, existing, externalOrderID, status, result)
		return
	}

	uc.createOne(ctx, in, wo, externalOrderID, status, result)
}

// routeToExisting sends an already-known order through UpdateSalesOrderUseCase
// (an unchanged status is a no-op) rather than duplicate its stock/status
// transition logic here -- shared by importOne's normal path and createOne's
// create-race fallback below.
func (uc *ImportWooOrdersUseCase) routeToExisting(ctx context.Context, existing *omsDomain.SalesOrder, externalOrderID string, status omsDomain.SalesOrderStatus, result *ImportOrdersResult) {
	if existing.Status == status {
		result.Unchanged++
		return
	}
	if _, err := uc.updateOrder.Execute(ctx, existing.ID, appoms.UpdateSalesOrderInput{Status: &status}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: update: %v", externalOrderID, err))
		return
	}
	result.Updated++
}

func (uc *ImportWooOrdersUseCase) createOne(ctx context.Context, in ImportWooOrdersInput, wo integrationsDomain.WooOrder, externalOrderID string, status omsDomain.SalesOrderStatus, result *ImportOrdersResult) {
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
		var customerID *string
		if wo.CustomerID != 0 {
			c, err := uc.customerRepo.FindByWooCustomerID(ctx, int64(wo.CustomerID))
			if err != nil {
				return err
			}
			if c != nil {
				customerID = &c.ID
			}
		}

		now := time.Now()
		order = omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceWooCommerce,
			ExternalOrderID: externalOrderID, ConnectionID: &in.ConnectionID, Number: wo.Number, OrderKey: wo.OrderKey,
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
			var skuID *string
			var taxClassID *string
			if li.SKU != "" {
				sku, err := uc.skuRepo.FindByCode(ctx, li.SKU)
				if err != nil && !errors.Is(err, pmsDomain.ErrSKUNotFound) {
					return err
				}
				if err == nil {
					skuID = &sku.ID
					taxClassID = sku.TaxClassID
				} else {
					unlinkedLines++
				}
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
		return nil
	})
	if err != nil {
		if errors.Is(err, omsDomain.ErrDuplicateExternalOrder) {
			// Lost a create race: another concurrent import of this same
			// order (a second scheduler replica, or the manual fetch-order
			// endpoint firing at the same moment) won it first. Route
			// through the same existing-order path importOne uses instead
			// of surfacing this as a failure -- the winner already created
			// (and, via UpdateSalesOrderUseCase's own conditional-write
			// guard, will safely decrement) the order.
			existing, findErr := uc.findExisting(ctx, externalOrderID)
			if findErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, findErr))
				return
			}
			if existing != nil {
				uc.routeToExisting(ctx, existing, externalOrderID, status, result)
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
	stockInsufficient := false
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
			stockInsufficient = true
			if err := setChannelsLineItemStockStatus(ctx, uc.db, uc.lineItemRepo, li.ID, omsDomain.StockStatusPending); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
			}
			continue
		}
		for _, movement := range movements {
			if err := linkOrderLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: link movement: %v", externalOrderID, err))
			}
		}
	}
	if stockInsufficient {
		if err := setChannelsOrderNeedsStockReview(ctx, uc.db, uc.salesOrderRepo, order.ID, true); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
		}
	}
}

// findExisting is FindByExternalID wrapped in its own tenant.WithTenant call
// -- used by createOne's create-race fallback, outside the (already failed
// and rolled back) transaction that attempted the original Create.
func (uc *ImportWooOrdersUseCase) findExisting(ctx context.Context, externalOrderID string) (*omsDomain.SalesOrder, error) {
	var existing *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		existing, err = uc.salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourceWooCommerce, externalOrderID)
		return err
	})
	return existing, err
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

func setChannelsOrderNeedsStockReview(ctx context.Context, db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, orderID string, needsReview bool) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, orderID, map[string]any{"needs_stock_review": needsReview})
	})
	if err != nil {
		return fmt.Errorf("setChannelsOrderNeedsStockReview: %w", err)
	}
	return nil
}

func wooAddressToOMS(a integrationsDomain.WooAddress) omsDomain.Address {
	return omsDomain.Address{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Email: a.Email, Phone: a.Phone,
	}
}

// parseWooGMTTime parses Woo's *_gmt date format ("2006-01-02T15:04:05",
// no offset -- see WooOrder's doc comment). Empty string errors, same as
// an unparseable value -- callers treat both as "no value".
func parseWooGMTTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("parseWooGMTTime: empty")
	}
	return time.Parse("2006-01-02T15:04:05", s)
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
