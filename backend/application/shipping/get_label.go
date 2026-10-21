package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetShipmentLabelUseCase fetches a shipment's label -- the stored one when
// its carrier returned it at creation, otherwise from the carrier, in the
// content format the shipment was created with (Shipment.LabelFormat --
// pdf/zpl, fixed at creation) -- the frontend's ShippingLabelPrintView
// renders whatever comes back.
type GetShipmentLabelUseCase struct {
	db             *gorm.DB
	shipmentRepo   shippingDomain.ShipmentRepository
	connRepo       integrationsDomain.ConnectionRepository
	settingsRepo   shippingDomain.ShippingConnectionSettingsRepository
	carrierFactory integrationsDomain.CarrierClientFactory
	encSecret      string
	labelRepo      shippingDomain.ShipmentLabelRepository
}

func NewGetShipmentLabelUseCase(
	db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	carrierFactory integrationsDomain.CarrierClientFactory,
	encryptionSecret string,
	labelRepo shippingDomain.ShipmentLabelRepository,
) *GetShipmentLabelUseCase {
	return &GetShipmentLabelUseCase{
		db: db, shipmentRepo: shipmentRepo, connRepo: connRepo, settingsRepo: settingsRepo,
		carrierFactory: carrierFactory, encSecret: encryptionSecret, labelRepo: labelRepo,
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
	var stored *shippingDomain.ShipmentLabel
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
		if stored, err = uc.labelRepo.FindByShipmentID(ctx, s.ID); err != nil {
			return err
		}
		settings, err = uc.settingsRepo.FindByConnectionID(ctx, s.ConnectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}
	// Carriers that hand the label over only at creation (PostNL) have it
	// stored; it's served as created, whatever size this print asks for.
	if stored != nil {
		return &integrationsDomain.LabelResult{Data: stored.Data, ContentType: stored.ContentType}, nil
	}

	if size == "" {
		size = shippingDomain.LabelSizeA4
		if settings != nil && settings.DefaultLabelSize != "" {
			size = settings.DefaultLabelSize
		}
	}

	client, err := buildCarrierClient(uc.carrierFactory, uc.encSecret, conn)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}

	label, err := client.GetLabel(ctx, shipment.ExternalShipmentID, shipment.LabelFormat, size)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}
	return &label, nil
}
