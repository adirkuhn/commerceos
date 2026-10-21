package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RefreshShipmentTrackingUseCase is the tracking timeline's "fetch more
// info from the shipping provider" button -- a staff-triggered version of
// what HandleShipmentWebhookUseCase does automatically on a webhook: ask
// the provider for this shipment's current status right now, and save
// whatever comes back as a new ShipmentStatusEvent (never just returned
// transiently) via the same applyStatusUpdate path webhook/reconciliation
// already share, so all three sources can never drift on what a status
// change actually does.
type RefreshShipmentTrackingUseCase struct {
	db              *gorm.DB
	shipmentRepo    shippingDomain.ShipmentRepository
	statusEventRepo shippingDomain.ShipmentStatusEventRepository
	connRepo        integrationsDomain.ConnectionRepository
	salesOrderRepo  omsDomain.SalesOrderRepository
	carrierFactory  integrationsDomain.CarrierClientFactory
	encSecret       string
}

func NewRefreshShipmentTrackingUseCase(
	db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	connRepo integrationsDomain.ConnectionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	carrierFactory integrationsDomain.CarrierClientFactory,
	encryptionSecret string,
) *RefreshShipmentTrackingUseCase {
	return &RefreshShipmentTrackingUseCase{
		db: db, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo, connRepo: connRepo,
		salesOrderRepo: salesOrderRepo, carrierFactory: carrierFactory, encSecret: encryptionSecret,
	}
}

func (uc *RefreshShipmentTrackingUseCase) Execute(ctx context.Context, shipmentID string) (shippingDomain.ShipmentStatusEvent, error) {
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
		return shippingDomain.ShipmentStatusEvent{}, fmt.Errorf("refreshShipmentTracking: %w", err)
	}

	client, err := buildCarrierClient(uc.carrierFactory, uc.encSecret, conn)
	if err != nil {
		return shippingDomain.ShipmentStatusEvent{}, fmt.Errorf("refreshShipmentTracking: %w", err)
	}

	tt, err := client.GetTrackTrace(ctx, shipment.ExternalShipmentID)
	if err != nil {
		return shippingDomain.ShipmentStatusEvent{}, fmt.Errorf("refreshShipmentTracking: get tracktrace: %w", err)
	}
	barcode := fetchBarcodeIfMissing(ctx, client, shipment)

	event, err := applyStatusUpdate(ctx, uc.db, uc.shipmentRepo, uc.statusEventRepo, uc.salesOrderRepo,
		shipment.ID, shipment.SalesOrderID, tt, shippingDomain.StatusEventSourceManual, barcode, nil)
	if err != nil {
		return shippingDomain.ShipmentStatusEvent{}, fmt.Errorf("refreshShipmentTracking: %w", err)
	}
	return event, nil
}
