package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

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
	// defaults -- nil/empty means "use the resolved default."
	PackageType               *shippingDomain.PackageType
	SignatureRequired         *bool
	OnlyRecipient             *bool
	LabelFormat               string
	InsuranceAmountMinorUnits *int
	// UserID is the acting staff member, for the activity log entry.
	UserID string
}

// CreateShipmentUseCase is the pick/pack "Create shipping label" action:
// resolves this connection's carrier options, creates the shipment with
// MyParcel, and persists the result. Registering the tenant's webhook
// callback is a separate explicit step (RegisterMyParcelWebhookUseCase,
// run once after connecting) -- not repeated here on every shipment.
type CreateShipmentUseCase struct {
	db              *gorm.DB
	connRepo        integrationsDomain.ConnectionRepository
	settingsRepo    shippingDomain.ShippingConnectionSettingsRepository
	overrideRepo    shippingDomain.ShippingCarrierOverrideRepository
	shipmentRepo    shippingDomain.ShipmentRepository
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
	recordActivity  recordActivityUC
}

func NewCreateShipmentUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	overrideRepo shippingDomain.ShippingCarrierOverrideRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
	recordActivity recordActivityUC,
) *CreateShipmentUseCase {
	return &CreateShipmentUseCase{
		db: db, connRepo: connRepo, settingsRepo: settingsRepo, overrideRepo: overrideRepo,
		shipmentRepo: shipmentRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
		recordActivity: recordActivity,
	}
}

func (uc *CreateShipmentUseCase) Execute(ctx context.Context, in CreateShipmentInput) (*shippingDomain.Shipment, error) {
	r := in.Recipient
	if r.Person == "" || r.Street == "" || r.City == "" || r.CC == "" {
		return nil, fmt.Errorf("createShipment: %w", shippingDomain.ErrMissingRecipient)
	}

	var conn integrationsDomain.Connection
	var settings *shippingDomain.ShippingConnectionSettings
	var override *shippingDomain.ShippingCarrierOverride
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
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("createShipment: %w", err)
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
	if in.InsuranceAmountMinorUnits != nil {
		resolved.InsuranceAmountMinorUnits = in.InsuranceAmountMinorUnits
		if *in.InsuranceAmountMinorUnits > 0 {
			resolved.SignatureRequired = true
			resolved.OnlyRecipient = true
		}
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("createShipment: decrypt api key: %w", err)
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("createShipment: build client: %w", err)
	}

	result, err := client.CreateShipment(ctx, integrationsDomain.CreateShipmentInput{
		Recipient: integrationsDomain.MyParcelAddress{
			Person: r.Person, Company: r.Company, Street: r.Street, Number: r.Number, NumberSuffix: r.NumberSuffix,
			PostalCode: r.PostalCode, City: r.City, CC: r.CC, Email: r.Email, Phone: r.Phone,
		},
		Carrier: int(in.Carrier),
		Options: integrationsDomain.MyParcelShipmentOptions{
			PackageType: int(resolved.PackageType), SignatureRequired: resolved.SignatureRequired, OnlyRecipient: resolved.OnlyRecipient,
			InsuranceAmountMinorEUR: resolved.InsuranceAmountMinorUnits,
		},
		OrderReference: in.SalesOrderID,
	})
	if err != nil {
		uc.recordShipmentActivity(ctx, in, "myparcel_error", "Failed to create MyParcel shipment", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("createShipment: %w", err)
	}

	shipment := shippingDomain.Shipment{
		SalesOrderID: in.SalesOrderID, ConnectionID: in.ConnectionID,
		Carrier: in.Carrier, MyParcelShipmentID: result.ShipmentID, Barcode: result.Barcode,
		Status: 0, StatusMain: "registered", LabelFormat: resolved.LabelFormat,
		PackageType: resolved.PackageType, SignatureRequired: resolved.SignatureRequired, OnlyRecipient: resolved.OnlyRecipient,
		InsuranceAmountMinorEUR: resolved.InsuranceAmountMinorUnits,
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
