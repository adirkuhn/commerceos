package shipping

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListShipmentTrackingEventsUseCase backs the Sales Order detail page's
// tracking timeline -- every status MyParcel has ever reported for one
// shipment, newest first, whichever of webhook/reconciliation/manual
// learned it.
type ListShipmentTrackingEventsUseCase struct {
	db              *gorm.DB
	statusEventRepo shippingDomain.ShipmentStatusEventRepository
}

func NewListShipmentTrackingEventsUseCase(db *gorm.DB, statusEventRepo shippingDomain.ShipmentStatusEventRepository) *ListShipmentTrackingEventsUseCase {
	return &ListShipmentTrackingEventsUseCase{db: db, statusEventRepo: statusEventRepo}
}

func (uc *ListShipmentTrackingEventsUseCase) Execute(ctx context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error) {
	var out []shippingDomain.ShipmentStatusEvent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.statusEventRepo.ListByShipmentID(ctx, shipmentID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listShipmentTrackingEvents: %w", err)
	}
	return out, nil
}
