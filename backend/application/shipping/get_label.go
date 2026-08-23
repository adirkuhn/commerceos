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
// format the shipment was created with (Shipment.LabelFormat) -- the
// frontend's ShippingLabelPrintView renders whatever comes back.
type GetShipmentLabelUseCase struct {
	db              *gorm.DB
	shipmentRepo    shippingDomain.ShipmentRepository
	connRepo        integrationsDomain.ConnectionRepository
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
}

func NewGetShipmentLabelUseCase(
	db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	connRepo integrationsDomain.ConnectionRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
) *GetShipmentLabelUseCase {
	return &GetShipmentLabelUseCase{
		db: db, shipmentRepo: shipmentRepo, connRepo: connRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
	}
}

func (uc *GetShipmentLabelUseCase) Execute(ctx context.Context, shipmentID string) (*integrationsDomain.LabelResult, error) {
	var shipment shippingDomain.Shipment
	var conn integrationsDomain.Connection
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: decrypt api key: %w", err)
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: build client: %w", err)
	}

	label, err := client.GetLabel(ctx, shipment.MyParcelShipmentID, shipment.LabelFormat)
	if err != nil {
		return nil, fmt.Errorf("getShipmentLabel: %w", err)
	}
	return &label, nil
}
