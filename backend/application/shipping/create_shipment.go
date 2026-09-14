package shipping

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// salesOrderTotalGetter is the one method CreateShipmentUseCase needs from
// oms.SalesOrderRepository (ISP) -- looking up the order's own value to
// insure it for, only when the resolved carrier/connection config says to.
type salesOrderTotalGetter interface {
	FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

// myParcelOrderExporter is *ExportMyParcelOrderUseCase narrowed to point of
// use (ISP) -- lets CreateShipmentUseCase optionally also register the
// order via MyParcel's separate Orders/Fulfilment API (visible on
// MyParcel's own Orders dashboard, distinct from the Shipments API this
// use case itself calls) when the connection's ExportOrderToMyParcel
// setting is on. ExportMyParcelOrderUseCase is idempotent by
// connection+sales order, so calling it here is safe even for a caller
// (e.g. the Sales Order detail page) that already called it explicitly
// first.
type myParcelOrderExporter interface {
	Execute(ctx context.Context, in ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error)
}

// skuWeightGetter is *pms.SKURepository narrowed to point of use (ISP) --
// CreateShipmentUseCase looks up each line item's own SKU to compute the
// shipment's total physical weight (MyParcel requires it for non-EU and
// digital-stamp shipments; see domain/integrations.CreateShipmentInput's
// doc comment).
type skuWeightGetter interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

// RecipientInput is the confirmed, MyParcel-shaped recipient address the
// frontend's address-confirm step collects -- see
// domain/shipping.Shipment's doc comment on why this isn't derived live
// from a SalesOrder's own free-text address.
type RecipientInput struct {
	Person       string
	Company      string
	Street       string
	Number       string
	NumberSuffix string
	PostalCode   string
	City         string
	CC           string
	Email        string
	Phone        string
}

type CreateShipmentInput struct {
	SalesOrderID string
	ConnectionID string
	Carrier      shippingDomain.Carrier
	Recipient    RecipientInput
	// PackageType/SignatureRequired/OnlyRecipient/LabelFormat are optional
	// per-shipment overrides on top of ResolveOptions' connection/carrier
	// defaults -- nil/empty means "use the resolved default." Insurance has
	// no such override here: whether to insure is entirely
	// ResolveOptions(settings, override, carrier).Insure, and the amount is
	// always this SalesOrder's own TotalMinorUnits, computed below -- never
	// a caller-supplied figure that could go stale against the order.
	PackageType       *shippingDomain.PackageType
	SignatureRequired *bool
	OnlyRecipient     *bool
	LabelFormat       string
	// UserID is the acting staff member, for the activity log entry.
	UserID string
}

// CreateShipmentUseCase is the pick/pack "Create shipping label" action:
// resolves this connection's carrier options, creates the shipment with
// MyParcel, and persists the result. Registering the tenant's webhook
// callback is a separate explicit step (RegisterMyParcelWebhookUseCase,
// run once after connecting) -- not repeated here on every shipment.
type CreateShipmentUseCase struct {
	db                 *gorm.DB
	connRepo           integrationsDomain.ConnectionRepository
	settingsRepo       shippingDomain.ShippingConnectionSettingsRepository
	overrideRepo       shippingDomain.ShippingCarrierOverrideRepository
	shipmentRepo       shippingDomain.ShipmentRepository
	carrierFactory     integrationsDomain.CarrierClientFactory
	encSecret          string
	recordActivity     recordActivityUC
	tenantSettingsRepo settingsDomain.Repository
	salesOrderRepo     salesOrderTotalGetter
	orderExporter      myParcelOrderExporter
	lineItemRepo       salesOrderLineItemLister
	skuRepo            skuWeightGetter
}

func NewCreateShipmentUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	overrideRepo shippingDomain.ShippingCarrierOverrideRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	carrierFactory integrationsDomain.CarrierClientFactory,
	encryptionSecret string,
	recordActivity recordActivityUC,
	tenantSettingsRepo settingsDomain.Repository,
	salesOrderRepo salesOrderTotalGetter,
	orderExporter myParcelOrderExporter,
	lineItemRepo salesOrderLineItemLister,
	skuRepo skuWeightGetter,
) *CreateShipmentUseCase {
	return &CreateShipmentUseCase{
		db: db, connRepo: connRepo, settingsRepo: settingsRepo, overrideRepo: overrideRepo,
		shipmentRepo: shipmentRepo, carrierFactory: carrierFactory, encSecret: encryptionSecret,
		recordActivity: recordActivity, tenantSettingsRepo: tenantSettingsRepo, salesOrderRepo: salesOrderRepo,
		orderExporter: orderExporter, lineItemRepo: lineItemRepo, skuRepo: skuRepo,
	}
}

func (uc *CreateShipmentUseCase) Execute(ctx context.Context, in CreateShipmentInput) (*shippingDomain.Shipment, error) {
	r := in.Recipient
	// Email is required, not just recommended -- MyParcel rejects some
	// carriers' shipments with a 422 ("recipient.email shorter than the
	// minimum length of 1") when it's blank, since MyParcel emails the
	// recipient tracking updates for those carriers. Checked here too
	// (not just required in the frontend form) so a stale/blank order
	// email can't silently reach MyParcel's API and fail there instead.
	if r.Person == "" || r.Street == "" || r.City == "" || r.CC == "" || r.Email == "" {
		return nil, fmt.Errorf("createShipment: %w", shippingDomain.ErrMissingRecipient)
	}

	var conn integrationsDomain.Connection
	var settings *shippingDomain.ShippingConnectionSettings
	var override *shippingDomain.ShippingCarrierOverride
	var tenantSettings *settingsDomain.TenantSettings
	var existing []shippingDomain.Shipment
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
		override, err = uc.overrideRepo.FindByConnectionAndCarrier(ctx, in.ConnectionID, in.Carrier)
		if err != nil {
			return err
		}
		tenantSettings, err = uc.tenantSettingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		existing, err = uc.shipmentRepo.FindBySalesOrderID(ctx, in.SalesOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("createShipment: %w", err)
	}
	// An active (non-cancelled) shipment already exists for this order on
	// this connection -- return it instead of creating a real second
	// MyParcel shipment. This is a defense-in-depth backstop (races, a
	// caller that skipped its own pre-check); the frontend's own "already
	// exists, re-download?" prompt is the primary UX for this (see
	// Picking.tsx/PickingScreen.tsx/SalesOrders.tsx), this just guarantees
	// the backend never silently duplicates regardless of what called it.
	// A cancelled shipment (CancelShipmentUseCase) is deliberately skipped
	// here -- that's exactly what unblocks creating a corrected shipment
	// after staff picked the wrong carrier.
	for _, s := range existing {
		if s.ConnectionID == in.ConnectionID && s.IsActive() {
			return &s, nil
		}
	}

	// Best-effort, non-blocking: exporting the order is purely for
	// visibility on MyParcel's own Orders dashboard, not required to
	// actually create the shipment/label -- a failure here must not stop
	// staff from getting their label.
	if settings != nil && settings.ExportOrderToMyParcel {
		if _, exportErr := uc.orderExporter.Execute(ctx, ExportMyParcelOrderInput{
			SalesOrderID: in.SalesOrderID, ConnectionID: in.ConnectionID, Carrier: in.Carrier,
			Recipient: in.Recipient, UserID: in.UserID,
		}); exportErr != nil {
			uc.recordShipmentActivity(ctx, in, "myparcel_order_export_failed",
				"Failed to export order to MyParcel (label creation continues)", map[string]any{"error": exportErr.Error()})
		}
	}

	resolved := shippingDomain.ResolveOptions(settings, override, in.Carrier)
	if in.PackageType != nil {
		resolved.PackageType = *in.PackageType
	}
	if in.SignatureRequired != nil {
		resolved.SignatureRequired = *in.SignatureRequired
	}
	if in.OnlyRecipient != nil {
		resolved.OnlyRecipient = *in.OnlyRecipient
	}
	if in.LabelFormat != "" {
		resolved.LabelFormat = in.LabelFormat
	}
	// Re-run the options gate: ResolveOptions already ran it, but only
	// against the connection/carrier-layered PackageType -- these
	// per-shipment overrides are layered on afterward, so a request that
	// picks a non-Package type per-shipment on top of a connection default
	// with signature/insurance on would otherwise bypass the gate entirely
	// (see GateOptionsForPackageType's doc comment).
	resolved = shippingDomain.GateOptionsForPackageType(resolved)

	if err := shippingDomain.ValidatePackageTypeForCarrier(in.Carrier, resolved.PackageType); err != nil {
		return nil, fmt.Errorf("createShipment: %w", err)
	}

	// The insured amount is always this specific order's own value, never a
	// caller-supplied or pre-configured figure -- looked up only when
	// actually insuring, to avoid the read otherwise.
	var insuranceAmount *int
	if resolved.Insure {
		var order *omsDomain.SalesOrder
		err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			var err error
			order, err = uc.salesOrderRepo.FindByID(ctx, in.SalesOrderID)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("createShipment: %w", err)
		}
		amt := int(order.TotalMinorUnits)
		insuranceAmount = &amt
	}

	weightGrams := uc.computeWeightGrams(ctx, in.SalesOrderID)

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("createShipment: decrypt api key: %w", err)
	}
	client, err := uc.carrierFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("createShipment: build client: %w", err)
	}

	result, err := client.CreateShipment(ctx, integrationsDomain.CreateCarrierShipmentRequest{
		Recipient: integrationsDomain.CarrierAddress{
			Person: r.Person, Company: r.Company, Street: r.Street, Number: r.Number, NumberSuffix: r.NumberSuffix,
			PostalCode: r.PostalCode, City: r.City, CC: r.CC, Email: r.Email, Phone: r.Phone,
		},
		Carrier: int(in.Carrier),
		Options: integrationsDomain.CarrierShipmentOptions{
			PackageType: int(resolved.PackageType), SignatureRequired: resolved.SignatureRequired, OnlyRecipient: resolved.OnlyRecipient,
			InsuranceAmountMinorUnits: insuranceAmount,
			Currency:                  tenantSettings.DefaultCurrency,
		},
		OrderReference: in.SalesOrderID,
		WeightGrams:    weightGrams,
	})
	if err != nil {
		uc.recordShipmentActivity(ctx, in, "myparcel_error", "Failed to create MyParcel shipment", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("createShipment: %w", err)
	}

	shipment := shippingDomain.Shipment{
		SalesOrderID: in.SalesOrderID, ConnectionID: in.ConnectionID,
		Carrier: in.Carrier, ExternalShipmentID: result.ExternalShipmentID, Barcode: result.Barcode,
		Status: 0, StatusMain: "registered", LabelFormat: resolved.LabelFormat,
		PackageType: resolved.PackageType, SignatureRequired: resolved.SignatureRequired, OnlyRecipient: resolved.OnlyRecipient,
		InsuranceAmountMinorEUR: insuranceAmount,
		RecipientPerson:         r.Person, RecipientCompany: r.Company, RecipientStreet: r.Street,
		RecipientNumber: r.Number, RecipientNumberSuffix: r.NumberSuffix, RecipientPostalCode: r.PostalCode,
		RecipientCity: r.City, RecipientCC: r.CC, RecipientEmail: r.Email, RecipientPhone: r.Phone,
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		shipment.TenantID = tenant.IDFromContext(ctx)
		return uc.shipmentRepo.Create(ctx, &shipment)
	})
	if err != nil {
		return nil, fmt.Errorf("createShipment: persist: %w", err)
	}
	uc.recordShipmentActivity(ctx, in, "myparcel_shipment_created", "Shipment created via MyParcel", map[string]any{"barcode": shipment.Barcode})
	return &shipment, nil
}

// errWeightUnknown is computeWeightGrams' own internal signal to stop
// early inside its tenant.WithTenant closure -- never returned to a
// caller, always swallowed into a nil total.
var errWeightUnknown = errors.New("shipping: weight unknown")

// computeWeightGrams sums quantity*SKU.WeightGrams across the order's line
// items -- best-effort, like recordShipmentActivity: any error, a line
// item with no SKU on file, or a SKU with no weight recorded makes the
// total unknown (nil, meaning physical_properties is omitted from the
// MyParcel request) rather than risk sending a wrong/partial figure.
func (uc *CreateShipmentUseCase) computeWeightGrams(ctx context.Context, salesOrderID string) *int {
	total := 0
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		for _, li := range lineItems {
			if li.SKUID == nil {
				return errWeightUnknown
			}
			sku, err := uc.skuRepo.FindByID(ctx, *li.SKUID)
			if err != nil {
				return err
			}
			if sku.WeightGrams == nil {
				return errWeightUnknown
			}
			total += int(*sku.WeightGrams) * int(li.Quantity)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return &total
}

// recordShipmentActivity is best-effort -- a failure recording the
// activity must not mask the actual create-shipment result, success or
// error, since the activity log is a record of what happened, not a
// precondition for it.
func (uc *CreateShipmentUseCase) recordShipmentActivity(ctx context.Context, in CreateShipmentInput, activityType, summary string, metadata map[string]any) {
	if uc.recordActivity == nil {
		return
	}
	_, _ = uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
		SalesOrderID: in.SalesOrderID, UserID: in.UserID, ActivityType: activityType, Summary: summary, Metadata: metadata,
	})
}
