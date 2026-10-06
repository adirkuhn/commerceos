package channels

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"time"

	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ImportBolOrdersInput drives ImportBolOrdersUseCase.Execute. Unlike Woo's
// equivalent, there's no ModifiedAfter cursor -- Bol's Orders API has no
// arbitrary "modified since" filter, only a status filter (confirmed
// against the real Retailer API v10 spec), so every run simply pages
// status=OPEN. That means a Bol order that leaves the OPEN set (fully
// shipped) between runs stops appearing here -- its CommerceOS SalesOrder
// stays "processing" rather than auto-advancing to "completed". This is a
// deliberate v1 scope limit (orders import only, per explicit request);
// a reconciliation pass (mirroring SyncEntityOrdersReconciliation, polling
// status=ALL) would close that gap if it's ever needed.
type ImportBolOrdersInput struct {
	ConnectionID       string
	DefaultWarehouseID string
	SkipStockMovements bool
}

// maxBolOrderImportPages mirrors maxOrderImportPages' reasoning -- generous
// enough that hitting it on a real store's first run should be rare, still
// reported via result.Errors rather than silently truncating if it ever is.
const maxBolOrderImportPages = 5000

// ImportBolOrdersUseCase pulls orders from Bol.com into domain/oms.SalesOrder
// -- the Bol counterpart to ImportWooOrdersUseCase, same idempotency
// (FindByExternalID(SalesOrderSourceBol, orderID)) and partial-failure
// tolerance, but simpler: no incremental cursor (see ImportBolOrdersInput's
// doc), no customer linking (Bol exposes no stable per-order customer
// identity to link against), and line items resolve by EAN
// (SKURepository.FindByBarcode) instead of Woo's SKU code, matching how
// PushProductToBolUseCase already treats SKU.Barcode as the EAN.
type ImportBolOrdersUseCase struct {
	db             *gorm.DB
	connRepo       integrationsDomain.ConnectionRepository
	bolFactory     integrationsDomain.BolClientFactory
	encSecret      string
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo        pmsDomain.SKURepository
	allocateStock  saleStockAllocator
	updateOrder    updateSalesOrderUC
	outbox         eventrecord.Enqueuer
}

func NewImportBolOrdersUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	bolFactory integrationsDomain.BolClientFactory,
	encryptionSecret string,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	linkRepo omsDomain.SalesOrderLineItemStockMovementRepository,
	skuRepo pmsDomain.SKURepository,
	allocateStock saleStockAllocator,
	updateOrder updateSalesOrderUC,
	outbox eventrecord.Enqueuer,
) *ImportBolOrdersUseCase {
	return &ImportBolOrdersUseCase{
		db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret,
		salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo,
		skuRepo: skuRepo, allocateStock: allocateStock, updateOrder: updateOrder,
		outbox: outbox,
	}
}

func (uc *ImportBolOrdersUseCase) connect(ctx context.Context, connectionID string) (integrationsDomain.BolClient, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *existing
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("importBolOrders: %w", err)
	}
	client, err := uc.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("importBolOrders: build client: %w", err)
	}
	return client, nil
}

func (uc *ImportBolOrdersUseCase) Execute(ctx context.Context, in ImportBolOrdersInput) (*ImportOrdersResult, error) {
	client, err := uc.connect(ctx, in.ConnectionID)
	if err != nil {
		return nil, err
	}

	result := ImportOrdersResult{}
	for page := 1; page <= maxBolOrderImportPages; page++ {
		orders, err := client.ListOrders(ctx, page, "OPEN")
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("page %d: %v", page, err))
			return &result, nil
		}
		if len(orders) == 0 {
			return &result, nil
		}
		for _, bo := range orders {
			uc.importOne(ctx, client, in, bo, &result)
		}
	}
	result.Errors = append(result.Errors, fmt.Sprintf("reached the %d-page import cap -- some orders may not have been imported this run", maxBolOrderImportPages))
	return &result, nil
}

// deriveBolOrderStatus maps Bol's per-item Quantity/QuantityShipped/
// QuantityCancelled facts (there's no single order-level status field on
// Bol's side) onto the existing SalesOrderStatus vocabulary -- every item
// fully shipped means completed; any item with a cancellation and nothing
// shipped yet means cancelled; anything else (still open, or a mixed
// partially-shipped/partially-cancelled order) means processing.
func deriveBolOrderStatus(items []integrationsDomain.BolOrderItem) omsDomain.SalesOrderStatus {
	allShipped := len(items) > 0
	anyCancelled := false
	anyShipped := false
	for _, it := range items {
		if it.QuantityShipped < it.Quantity {
			allShipped = false
		}
		if it.QuantityShipped > 0 {
			anyShipped = true
		}
		if it.QuantityCancelled > 0 {
			anyCancelled = true
		}
	}
	if allShipped {
		return omsDomain.SalesOrderStatusCompleted
	}
	if anyCancelled && !anyShipped {
		return omsDomain.SalesOrderStatusCancelled
	}
	return omsDomain.SalesOrderStatusProcessing
}

func (uc *ImportBolOrdersUseCase) importOne(ctx context.Context, client integrationsDomain.BolClient, in ImportBolOrdersInput, bo integrationsDomain.BolOrder, result *ImportOrdersResult) {
	status := deriveBolOrderStatus(bo.OrderItems)

	var existing *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		existing, err = uc.salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourceBol, bo.OrderID)
		return err
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: %v", bo.OrderID, err))
		return
	}
	if existing != nil {
		uc.routeToExisting(ctx, existing, bo.OrderID, status, result)
		return
	}

	full, err := client.GetOrder(ctx, bo.OrderID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: fetch detail: %v", bo.OrderID, err))
		return
	}
	uc.createOne(ctx, in, *full, bo.OrderID, status, result)
}

func (uc *ImportBolOrdersUseCase) routeToExisting(ctx context.Context, existing *omsDomain.SalesOrder, externalOrderID string, status omsDomain.SalesOrderStatus, result *ImportOrdersResult) {
	if existing.Status == status {
		result.Unchanged++
		return
	}
	if _, err := uc.updateOrder.Execute(ctx, existing.ID, appoms.UpdateSalesOrderInput{Status: &status}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: update: %v", externalOrderID, err))
		return
	}
	result.Updated++
}

func bolAddressToOMS(a integrationsDomain.BolShipmentDetails) omsDomain.Address {
	return omsDomain.Address{
		FirstName: a.FirstName, LastName: a.Surname,
		Address1: a.StreetName + " " + a.HouseNumber, City: a.City,
		Postcode: a.ZipCode, Country: a.CountryCode, Email: a.Email,
	}
}

// The customer pays bol.com, which pays the store out later -- a bol order
// has no payment transaction of its own, it's simply paid via bol.com.
const (
	bolPaymentMethod      = "bol"
	bolPaymentMethodTitle = "bol.com"
)

func (uc *ImportBolOrdersUseCase) createOne(ctx context.Context, in ImportBolOrdersInput, bo integrationsDomain.BolOrder, externalOrderID string, status omsDomain.SalesOrderStatus, result *ImportOrdersResult) {
	occurredAt := bo.OrderPlacedAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	stockReducing := status.ReducesStock() && !in.SkipStockMovements
	address := bolAddressToOMS(bo.ShipmentDetails)

	var totalMinorUnits int64
	for _, li := range bo.OrderItems {
		if li.Quantity > 0 {
			totalMinorUnits += li.UnitPriceMinorUnits * int64(li.Quantity)
		}
	}

	var order omsDomain.SalesOrder
	var createdLineItems []omsDomain.SalesOrderLineItem
	skippedLines := 0
	unlinkedLines := 0
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		order = omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceBol,
			ExternalOrderID: externalOrderID, ConnectionID: &in.ConnectionID, Number: externalOrderID,
			Status: status, WarehouseID: in.DefaultWarehouseID,
			Billing: address, Shipping: address, CurrencyCode: "EUR", TotalMinorUnits: totalMinorUnits,
			PaymentMethod: bolPaymentMethod, PaymentMethodTitle: bolPaymentMethodTitle,
			OccurredAt: occurredAt, CreatedAt: now, UpdatedAt: now,
			StockDecremented: stockReducing,
			Fulfilled:        status == omsDomain.SalesOrderStatusCompleted,
		}
		if err := uc.salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		for _, li := range bo.OrderItems {
			if li.Quantity <= 0 {
				skippedLines++
				continue
			}
			var skuID *string
			var taxClassID *string
			if li.EAN != "" {
				sku, err := uc.skuRepo.FindByBarcode(ctx, li.EAN)
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
			quantity := int64(li.Quantity)
			lineTotal := li.UnitPriceMinorUnits * quantity

			stockStatus := omsDomain.StockStatusDecremented
			if stockReducing && skuID == nil {
				stockStatus = omsDomain.StockStatusPending
			}
			lineItem := omsDomain.SalesOrderLineItem{
				TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: skuID,
				Name: li.Title, Barcode: li.EAN, Quantity: quantity, StockStatus: stockStatus,
				PriceMinorUnits: li.UnitPriceMinorUnits, SubtotalMinorUnits: lineTotal,
				TotalMinorUnits: lineTotal, TaxClassID: taxClassID, CreatedAt: now,
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
		return recordOrderImported(ctx, uc.outbox, order, len(createdLineItems), unlinkedLines)
	})
	if err != nil {
		if errors.Is(err, omsDomain.ErrDuplicateExternalOrder) {
			existing, findErr := uc.findExisting(ctx, externalOrderID)
			if findErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: %v", externalOrderID, findErr))
				return
			}
			if existing != nil {
				uc.routeToExisting(ctx, existing, externalOrderID, status, result)
				return
			}
		}
		result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: %v", externalOrderID, err))
		return
	}
	result.Created++
	if skippedLines > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: %d line item(s) skipped (non-positive quantity)", externalOrderID, skippedLines))
	}

	if !stockReducing {
		return
	}
	var pendingLineItemIDs []string
	for _, li := range createdLineItems {
		if li.SKUID == nil {
			continue
		}
		movements, err := uc.allocateStock.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: in.DefaultWarehouseID, Type: invDomain.MovementTypeSale,
			Quantity: li.Quantity, Reason: fmt.Sprintf("bol order %s", externalOrderID), OccurredAt: occurredAt,
		})
		if err != nil {
			if !errors.Is(err, invDomain.ErrInsufficientStock) {
				result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: stock movement: %v", externalOrderID, err))
				continue
			}
			if err := setChannelsLineItemStockStatus(ctx, uc.db, uc.lineItemRepo, li.ID, omsDomain.StockStatusPending); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: %v", externalOrderID, err))
				continue
			}
			pendingLineItemIDs = append(pendingLineItemIDs, li.ID)
			continue
		}
		for _, movement := range movements {
			if err := linkOrderLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: link movement: %v", externalOrderID, err))
			}
		}
	}
	if len(pendingLineItemIDs) > 0 {
		if err := flagImportedOrderStockShortage(ctx, uc.db, uc.salesOrderRepo, uc.outbox, order.ID, pendingLineItemIDs); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bol order %s: %v", externalOrderID, err))
		}
	}
}

func (uc *ImportBolOrdersUseCase) findExisting(ctx context.Context, externalOrderID string) (*omsDomain.SalesOrder, error) {
	var existing *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		existing, err = uc.salesOrderRepo.FindByExternalID(ctx, omsDomain.SalesOrderSourceBol, externalOrderID)
		return err
	})
	return existing, err
}
