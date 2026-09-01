package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListShipmentsForOrderUseCase backs the pick/pack screen's "does this
// order already have a shipment/tracking code" check -- newest first. Also
// resolves each shipment's best available tracking link (the connection's
// configured carrier override template, if any, else whatever MyParcel's
// own tracktrace response already gave it) before returning, so callers
// (the Sales Order detail page, the Woo shipment-info note) never need
// their own copy of that resolution logic.
type ListShipmentsForOrderUseCase struct {
	db                  *gorm.DB
	shipmentRepo        shippingDomain.ShipmentRepository
	carrierOverrideRepo shippingDomain.ShippingCarrierOverrideRepository
}

func NewListShipmentsForOrderUseCase(db *gorm.DB, shipmentRepo shippingDomain.ShipmentRepository, carrierOverrideRepo shippingDomain.ShippingCarrierOverrideRepository) *ListShipmentsForOrderUseCase {
	return &ListShipmentsForOrderUseCase{db: db, shipmentRepo: shipmentRepo, carrierOverrideRepo: carrierOverrideRepo}
}

func (uc *ListShipmentsForOrderUseCase) Execute(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
	var result []shippingDomain.Shipment
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.shipmentRepo.FindBySalesOrderID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		for i := range result {
			override, err := uc.carrierOverrideRepo.FindByConnectionAndCarrier(ctx, result[i].ConnectionID, result[i].Carrier)
			if err != nil {
				return err
			}
			if override == nil {
				continue
			}
			if link := shippingDomain.BuildCarrierTrackingURL(override.TrackingURLTemplate, result[i].Barcode, result[i].RecipientPostalCode); link != "" {
				result[i].TrackingURL = link
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listShipmentsForOrder: %w", err)
	}
	return result, nil
}
