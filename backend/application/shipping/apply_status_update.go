package shipping

import (
	"context"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// applyStatusUpdate is the one place a fetched MyParcel TrackTraceResult
// turns into local writes -- shared by HandleShipmentWebhookUseCase
// (webhook-triggered) and ReconcileShipmentsUseCase (poll-triggered) so the
// two never drift on what a status change actually does (record history +
// the one Fulfilled-on-"shipped" order effect).
func applyStatusUpdate(
	ctx context.Context, db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	shipmentID, salesOrderID string,
	tt integrationsDomain.TrackTraceResult,
	source shippingDomain.StatusEventSource,
) error {
	return tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := shipmentRepo.UpdateStatus(ctx, shipmentID, tt.StatusCode, tt.StatusMain); err != nil {
			return err
		}
		if err := statusEventRepo.Append(ctx, &shippingDomain.ShipmentStatusEvent{
			TenantID: tenant.IDFromContext(ctx), ShipmentID: shipmentID, Status: tt.StatusCode, StatusMain: tt.StatusMain,
			Source: source, OccurredAt: time.Now(),
		}); err != nil {
			return err
		}
		if tt.StatusMain == shippingDomain.StatusMainShipped {
			if err := salesOrderRepo.Update(ctx, salesOrderID, map[string]any{"fulfilled": true, "updated_at": time.Now()}); err != nil {
				return err
			}
		}
		return nil
	})
}
