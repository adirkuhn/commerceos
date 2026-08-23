package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListShipmentsForOrderUseCase backs the pick/pack screen's "does this
// order already have a shipment/tracking code" check -- newest first.
type ListShipmentsForOrderUseCase struct {
	db           *gorm.DB
	shipmentRepo shippingDomain.ShipmentRepository
}

func NewListShipmentsForOrderUseCase(db *gorm.DB, shipmentRepo shippingDomain.ShipmentRepository) *ListShipmentsForOrderUseCase {
	return &ListShipmentsForOrderUseCase{db: db, shipmentRepo: shipmentRepo}
}

func (uc *ListShipmentsForOrderUseCase) Execute(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
	var result []shippingDomain.Shipment
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result, err = uc.shipmentRepo.FindBySalesOrderID(ctx, salesOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listShipmentsForOrder: %w", err)
	}
	return result, nil
}
