package shipping

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// recordActivityUC is *application/oms.RecordSalesOrderActivityUseCase,
// narrowed to point of use (ISP) -- lets this use case write to the OMS
// activity log without importing domain/oms's repository directly, per
// CLAUDE.md's cross-domain-calls-through-use-cases rule. Same pattern as
// application/oms's own recordSaleMovementUC.
type recordActivityUC interface {
	Execute(ctx context.Context, in appoms.RecordSalesOrderActivityInput) (*omsDomain.SalesOrderActivity, error)
}

// salesOrderLineItemLister is *oms.SalesOrderLineItemRepository narrowed to
// point of use (ISP) -- ExportMyParcelOrderUseCase needs the order's own
// line items to build MyParcel's required order_lines[] array.
type salesOrderLineItemLister interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error)
}

type ExportMyParcelOrderInput struct {
	SalesOrderID string
	ConnectionID string
	Carrier      shippingDomain.Carrier
	Recipient    RecipientInput
	// UserID is the acting staff member, for the activity log entry.
	UserID string
}

// ExportMyParcelOrderUseCase is step one of the Sales Order detail page's
// "Create label" flow: registers the order with MyParcel's Orders API
// before the user confirms the address and CreateShipmentUseCase actually
// creates the shipment (see CreateOrderInput's doc for why this is a
// separate real MyParcel endpoint, not folded into CreateShipmentUseCase).
// Idempotent: re-calling for an already-exported order returns the
// existing export instead of creating a duplicate MyParcel order.
type ExportMyParcelOrderUseCase struct {
	db              *gorm.DB
	connRepo        integrationsDomain.ConnectionRepository
	settingsRepo    shippingDomain.ShippingConnectionSettingsRepository
	fulfilmentRepo  shippingDomain.MyParcelFulfilmentOrderRepository
	salesOrderRepo  salesOrderTotalGetter
	lineItemRepo    salesOrderLineItemLister
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
	recordActivity  recordActivityUC
	outbox          eventrecord.Enqueuer
}

func NewExportMyParcelOrderUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	fulfilmentRepo shippingDomain.MyParcelFulfilmentOrderRepository,
	salesOrderRepo salesOrderTotalGetter,
	lineItemRepo salesOrderLineItemLister,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
	recordActivity recordActivityUC,
	outbox eventrecord.Enqueuer,
) *ExportMyParcelOrderUseCase {
	return &ExportMyParcelOrderUseCase{
		db: db, connRepo: connRepo, settingsRepo: settingsRepo, fulfilmentRepo: fulfilmentRepo,
		salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo,
		myParcelFactory: myParcelFactory, encSecret: encryptionSecret, recordActivity: recordActivity, outbox: outbox,
	}
}

func (uc *ExportMyParcelOrderUseCase) Execute(ctx context.Context, in ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	r := in.Recipient
	// See CreateShipmentUseCase's identical check -- email is required, not
	// just recommended.
	if r.Person == "" || r.Street == "" || r.City == "" || r.CC == "" || r.Email == "" {
		return nil, fmt.Errorf("exportMyParcelOrder: %w", shippingDomain.ErrMissingRecipient)
	}

	var conn integrationsDomain.Connection
	var settings *shippingDomain.ShippingConnectionSettings
	var existing *shippingDomain.MyParcelFulfilmentOrder
	var order *omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		settings, err = uc.settingsRepo.FindByConnectionID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		existing, err = uc.fulfilmentRepo.FindByConnectionAndSalesOrder(ctx, in.ConnectionID, in.SalesOrderID)
		if err != nil || existing != nil {
			// Already exported -- skip the order/line-item fetch below
			// entirely, nothing here will be used to build a request.
			return err
		}
		order, err = uc.salesOrderRepo.FindByID(ctx, in.SalesOrderID)
		if err != nil {
			return err
		}
		lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, in.SalesOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("exportMyParcelOrder: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	resolved := shippingDomain.ResolveOptions(settings, nil, in.Carrier)
	accountID, shopID := 0, 0
	if settings != nil && settings.MyParcelAccountID != nil {
		accountID = *settings.MyParcelAccountID
	}
	if settings != nil && settings.MyParcelShopID != nil {
		shopID = *settings.MyParcelShopID
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("exportMyParcelOrder: decrypt api key: %w", err)
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("exportMyParcelOrder: build client: %w", err)
	}

	// externalIdentifier is the order's human-facing number, not our own
	// internal SalesOrder.ID -- MyParcel's Orders API has no separate
	// per-shipment reference field (see infrastructure/myparcel's package
	// doc comment), this external_identifier is the only reference visible
	// on MyParcel's own dashboard for the order. Falls back to the internal
	// ID only in the unexpected case Number is blank.
	externalIdentifier := order.Number
	if externalIdentifier == "" {
		externalIdentifier = in.SalesOrderID
	}

	orderLines := make([]integrationsDomain.OrderLineInput, len(lineItems))
	for i, li := range lineItems {
		orderLines[i] = integrationsDomain.OrderLineInput{
			Quantity: int(li.Quantity), SKU: li.SKUCode, Name: li.Name,
			PriceMinorUnits: int(li.TotalMinorUnits), VATMinorUnits: int(li.TotalTaxMinorUnits),
			PriceAfterVATMinorUnits: int(li.TotalMinorUnits + li.TotalTaxMinorUnits),
		}
	}
	priceMinorUnits := int(order.TotalMinorUnits - order.TaxTotalMinorUnits)

	result, err := client.CreateOrder(ctx, integrationsDomain.CreateOrderInput{
		ExternalIdentifier: externalIdentifier, OrderDate: time.Now(), AccountID: accountID, ShopID: shopID,
		Carrier: int(in.Carrier), PackageType: int(resolved.PackageType),
		Recipient: integrationsDomain.MyParcelAddress{
			Person: r.Person, Company: r.Company, Street: r.Street, Number: r.Number, NumberSuffix: r.NumberSuffix,
			PostalCode: r.PostalCode, City: r.City, CC: r.CC, Email: r.Email, Phone: r.Phone,
		},
		PriceMinorUnits: priceMinorUnits, VATMinorUnits: int(order.TaxTotalMinorUnits), PriceAfterVATMinorUnits: int(order.TotalMinorUnits),
		OrderLines: orderLines,
	})
	if err != nil {
		uc.recordFailure(ctx, in, err)
		return nil, fmt.Errorf("exportMyParcelOrder: %w", err)
	}

	fulfilmentOrder := shippingDomain.MyParcelFulfilmentOrder{
		ConnectionID: in.ConnectionID, SalesOrderID: in.SalesOrderID, MyParcelUUID: result.UUID, Status: result.Status,
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		fulfilmentOrder.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		fulfilmentOrder.CreatedAt, fulfilmentOrder.UpdatedAt = now, now
		if err := uc.fulfilmentRepo.Create(ctx, &fulfilmentOrder); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.FulfilmentOrderExported, fulfilmentOrder.ID, eventrecord.Data{
			"sales_order_id": in.SalesOrderID, "connection_id": in.ConnectionID,
			"myparcel_uuid": fulfilmentOrder.MyParcelUUID, "status": fulfilmentOrder.Status,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("exportMyParcelOrder: persist: %w", err)
	}

	if uc.recordActivity != nil {
		if _, err := uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
			SalesOrderID: in.SalesOrderID, UserID: in.UserID, ActivityType: "myparcel_order_exported",
			Summary:  "Order exported to MyParcel",
			Metadata: map[string]any{"myparcel_uuid": result.UUID, "status": result.Status},
		}); err != nil {
			log.Printf("exportMyParcelOrder: record activity for %s: %v", in.SalesOrderID, err)
		}
	}
	return &fulfilmentOrder, nil
}

// recordFailure writes an activity entry for a MyParcel API error -- a
// separate, always-attempted call after the failed external request,
// since the failure itself can't live inside a DB transaction that never
// committed. Best-effort: an error recording the failure must not mask
// the original MyParcel error returned to the caller.
func (uc *ExportMyParcelOrderUseCase) recordFailure(ctx context.Context, in ExportMyParcelOrderInput, cause error) {
	if uc.recordActivity == nil {
		return
	}
	if _, err := uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
		SalesOrderID: in.SalesOrderID, UserID: in.UserID, ActivityType: "myparcel_error",
		Summary:  "Failed to export order to MyParcel",
		Metadata: map[string]any{"error": cause.Error()},
	}); err != nil {
		log.Printf("exportMyParcelOrder: record error activity for %s: %v", in.SalesOrderID, err)
	}
}
