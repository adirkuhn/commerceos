package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetShipmentLabelUseCase fetches a shipment's label from MyParcel, in the
// content format the shipment was created with (Shipment.LabelFormat --
// pdf/zpl, fixed at creation) -- the frontend's ShippingLabelPrintView
// renders whatever comes back.
type GetShipmentLabelUseCase struct {
	db              *gorm.DB
	shipmentRepo    shippingDomain.ShipmentRepository
	connRepo        integrationsDomain.ConnectionRepository
	settingsRepo    shippingDomain.ShippingConnectionSettingsRepository
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
}

func NewGetShipmentLabelUseCase(
	db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
) *GetShipmentLabelUseCase {
	return &GetShipmentLabelUseCase{
		db: db, shipmentRepo: shipmentRepo, connRepo: connRepo, settingsRepo: settingsRepo,
		myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
	}
}

// Execute fetches shipmentID's label. size (LabelSizeA4/LabelSizeA6) is the
// physical page format requested for THIS print -- unlike LabelFormat, this
// is a retrieval-time choice, not fixed when the shipment was created, so
// staff can print/reprint the same label at a different size. Empty falls
// back to the connection's DefaultLabelSize, itself defaulting to
// LabelSizeA4 when never configured.
func (uc *GetShipmentLabelUseCase) Execute(ctx context.Context, shipmentID, size string) (*integrationsDomain.LabelResult, error) {
	var shipment shippingDomain.Shipment
	var conn integrationsDomain.Connection
	var settings *shippingDomain.ShippingConnectionSettings
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.shipmentRepo.FindByID(ctx, shipmentID)
		if err != nil {
			return err
		}
		shipment = *s
		c, err := uc.connRepo.FindByID(ctx, s.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		settings, err = uc.settingsRepo.FindByConnectionID(ctx, s.ConnectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}

	if size == "" {
		size = shippingDomain.LabelSizeA4
		if settings != nil && settings.DefaultLabelSize != "" {
			size = settings.DefaultLabelSize
		}
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: decrypt api key: %w", err)
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: build client: %w", err)
	}

	label, err := client.GetLabel(ctx, shipment.MyParcelShipmentID, shipment.LabelFormat, size)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}
	return &label, nil
}
