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
// comparatively bounded.
const maxOrderImportPages = 500

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
	stock            stockRecorder
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
	stock stockRecorder,
	updateOrder updateSalesOrderUC,
) *ImportWooOrdersUseCase {
	return &ImportWooOrdersUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret,
		salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo,
		skuRepo: skuRepo, customerRepo: customerRepo, stock: stock, updateOrder: updateOrder,
	}
}

func (uc *ImportWooOrdersUseCase) Execute(ctx context.Context, in ImportWooOrdersInput) (*ImportOrdersResult, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *existing
		secret, err = crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("importWooOrders: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("importWooOrders: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	result := ImportOrdersResult{}
	for page := 1; page <= maxOrderImportPages; page++ {
		orders, err := client.ListOrders(ctx, page, 50, in.ModifiedAfter)
		if err != nil {
			return nil, fmt.Errorf("importWooOrders: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}
		if len(orders) == 0 {
			break
		}
		for _, wo := range orders {
			uc.importOne(ctx, in, wo, &result)
		}
	}
	return &result, nil
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
		if existing.Status == status {
			result.Unchanged++
			return
		}
		if _, err := uc.updateOrder.Execute(ctx, existing.ID, appoms.UpdateSalesOrderInput{Status: &status}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: update: %v", externalOrderID, err))
			return
		}
		result.Updated++
		return
	}

	uc.createOne(ctx, in, wo, externalOrderID, status, result)
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
	wooID := int64(wo.ID)

	var order omsDomain.SalesOrder
	var createdLineItems []omsDomain.SalesOrderLineItem
	skippedLines := 0
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
			ExternalOrderID: externalOrderID, WooOrderID: &wooID, Number: wo.Number, OrderKey: wo.OrderKey,
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
		}
		if err := uc.salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		for _, li := range wo.LineItems {
			if li.SKU == "" {
				skippedLines++
				continue
			}
			sku, err := uc.skuRepo.FindByCode(ctx, li.SKU)
			if err != nil {
				if errors.Is(err, pmsDomain.ErrSKUNotFound) {
					skippedLines++
					continue
				}
				return err
			}
			price, _ := ParseWooPriceToMinorUnits(li.Price)
			subtotal, _ := ParseWooPriceToMinorUnits(li.Subtotal)
			subtotalTax, _ := ParseWooPriceToMinorUnits(li.SubtotalTax)
			total, _ := ParseWooPriceToMinorUnits(li.Total)
			totalTax, _ := ParseWooPriceToMinorUnits(li.TotalTax)
			lineItem := omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: sku.ID,
				Name: li.Name, SKUCode: li.SKU, Quantity: li.Quantity,
				PriceMinorUnits: price, SubtotalMinorUnits: subtotal, SubtotalTaxMinorUnits: subtotalTax,
				TotalMinorUnits: total, TotalTaxMinorUnits: totalTax, CreatedAt: now,
			}
			if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
				return err
			}
			createdLineItems = append(createdLineItems, lineItem)
		}
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %v", externalOrderID, err))
		return
	}
	result.Created++
	if skippedLines > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: %d line item(s) skipped (no matching SKU in CommerceOS)", externalOrderID, skippedLines))
	}

	if !stockReducing {
		return
	}
	for _, li := range createdLineItems {
		movement, err := uc.stock.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: li.SKUID, WarehouseID: in.DefaultWarehouseID, Type: invDomain.MovementTypeSale,
			QuantityDelta: -li.Quantity, Reason: fmt.Sprintf("woo order %s", externalOrderID), OccurredAt: occurredAt,
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: stock movement: %v", externalOrderID, err))
			continue
		}
		if err := linkOrderLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: link movement: %v", externalOrderID, err))
		}
	}
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
func linkOrderLineItemMovement(ctx context.Context, db *gorm.DB, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, lineItemID, movementID string) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return linkRepo.Create(ctx, &omsDomain.SalesOrderLineItemStockMovement{
			TenantID: tenant.IDFromContext(ctx), SalesOrderLineItemID: lineItemID,
			StockMovementID: movementID, CreatedAt: time.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("linkOrderLineItemMovement: %w", err)
	}
	return nil
}
