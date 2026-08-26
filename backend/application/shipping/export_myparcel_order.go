package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
	recordActivity  recordActivityUC
}

func NewExportMyParcelOrderUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	fulfilmentRepo shippingDomain.MyParcelFulfilmentOrderRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
	recordActivity recordActivityUC,
) *ExportMyParcelOrderUseCase {
	return &ExportMyParcelOrderUseCase{
		db: db, connRepo: connRepo, settingsRepo: settingsRepo, fulfilmentRepo: fulfilmentRepo,
		myParcelFactory: myParcelFactory, encSecret: encryptionSecret, recordActivity: recordActivity,
	}
}

func (uc *ExportMyParcelOrderUseCase) Execute(ctx context.Context, in ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	r := in.Recipient
	if r.Person == "" || r.Street == "" || r.City == "" || r.CC == "" {
		return nil, fmt.Errorf("exportMyParcelOrder: %w", shippingDomain.ErrMissingRecipient)
	}

	var conn integrationsDomain.Connection
	var settings *shippingDomain.ShippingConnectionSettings
	var existing *shippingDomain.MyParcelFulfilmentOrder
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

	result, err := client.CreateOrder(ctx, integrationsDomain.CreateOrderInput{
		ExternalIdentifier: in.SalesOrderID, OrderDate: time.Now(), AccountID: accountID, ShopID: shopID,
		Carrier: int(in.Carrier), PackageType: int(resolved.PackageType),
		Recipient: integrationsDomain.MyParcelAddress{
			Person: r.Person, Company: r.Company, Street: r.Street, Number: r.Number, NumberSuffix: r.NumberSuffix,
			PostalCode: r.PostalCode, City: r.City, CC: r.CC, Email: r.Email, Phone: r.Phone,
		},
	})
	if err != nil {
		uc.recordFailure(ctx, in, err)
		return nil, fmt.Errorf("exportMyParcelOrder: %w", err)
	}

	order := shippingDomain.MyParcelFulfilmentOrder{
		ConnectionID: in.ConnectionID, SalesOrderID: in.SalesOrderID, MyParcelUUID: result.UUID, Status: result.Status,
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		order.CreatedAt, order.UpdatedAt = now, now
		return uc.fulfilmentRepo.Create(ctx, &order)
	})
	if err != nil {
		return nil, fmt.Errorf("exportMyParcelOrder: persist: %w", err)
	}

	if uc.recordActivity != nil {
		_, _ = uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
			SalesOrderID: in.SalesOrderID, UserID: in.UserID, ActivityType: "myparcel_order_exported",
			Summary:  "Order exported to MyParcel",
			Metadata: map[string]any{"myparcel_uuid": result.UUID, "status": result.Status},
		})
	}
	return &order, nil
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
	_, _ = uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
		SalesOrderID: in.SalesOrderID, UserID: in.UserID, ActivityType: "myparcel_error",
		Summary:  "Failed to export order to MyParcel",
		Metadata: map[string]any{"error": cause.Error()},
	})
}
