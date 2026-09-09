package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CancelShipmentInput struct {
	ShipmentID string
	// UserID is the acting staff member, for the activity log entry.
	UserID string
}

// CancelShipmentResult reports what actually happened to the shipment on
// MyParcel's side, distinct from the (always successful, unless an error is
// returned) local cancellation -- see CancelShipmentUseCase's doc comment
// for why these can diverge.
type CancelShipmentResult struct {
	DeletedOnMyParcel bool
	Message           string
}

// CancelShipmentUseCase is the "wrong carrier, start over" action: attempts
// MyParcel's real DELETE /shipments/{id}, then marks the shipment cancelled
// in CommerceOS regardless of that outcome, unblocking a corrected shipment
// for the same order.
//
// MyParcel only allows deleting a shipment before its label has been
// created (status 1, "pending - concept") -- since this app's own
// CreateShipmentUseCase creates the label as part of shipment creation, the
// MyParcel call here is expected to often be rejected (a 422). That
// rejection is not treated as a use case failure: the local cancellation
// (CommerceOS's own gate on creating a new shipment for this order) always
// proceeds, and CancelShipmentResult tells the caller plainly whether
// MyParcel's own copy was actually removed or merely superseded locally, so
// staff know a wrong-carrier label might still exist/trigger tracking
// events on MyParcel's side.
type CancelShipmentUseCase struct {
	db             *gorm.DB
	connRepo       integrationsDomain.ConnectionRepository
	shipmentRepo   shippingDomain.ShipmentRepository
	carrierFactory integrationsDomain.CarrierClientFactory
	encSecret      string
	recordActivity recordActivityUC
	now            func() time.Time
}

func NewCancelShipmentUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	carrierFactory integrationsDomain.CarrierClientFactory,
	encryptionSecret string,
	recordActivity recordActivityUC,
) *CancelShipmentUseCase {
	return &CancelShipmentUseCase{
		db: db, connRepo: connRepo, shipmentRepo: shipmentRepo,
		carrierFactory: carrierFactory, encSecret: encryptionSecret,
		recordActivity: recordActivity, now: time.Now,
	}
}

func (uc *CancelShipmentUseCase) Execute(ctx context.Context, in CancelShipmentInput) (*CancelShipmentResult, error) {
	var shipment *shippingDomain.Shipment
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.shipmentRepo.FindByID(ctx, in.ShipmentID)
		if err != nil {
			return err
		}
		shipment = s
		c, err := uc.connRepo.FindByID(ctx, s.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cancelShipment: %w", err)
	}

	if !shipment.IsActive() {
		return &CancelShipmentResult{DeletedOnMyParcel: false, Message: shipment.CancelNote}, nil
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("cancelShipment: decrypt api key: %w", err)
	}
	client, err := uc.carrierFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("cancelShipment: build client: %w", err)
	}

	deleteErr := client.CancelShipment(ctx, shipment.ExternalShipmentID)
	result := &CancelShipmentResult{}
	if deleteErr == nil {
		result.DeletedOnMyParcel = true
		result.Message = "Shipment deleted on MyParcel."
	} else {
		result.DeletedOnMyParcel = false
		result.Message = fmt.Sprintf(
			"MyParcel could not delete this shipment (%s). It has been cancelled in CommerceOS so a corrected shipment can be created, but the old shipment/label may still exist on MyParcel's own side.",
			deleteErr.Error(),
		)
	}

	cancelledAt := uc.now()
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.shipmentRepo.MarkCancelled(ctx, shipment.ID, cancelledAt, result.Message)
	})
	if err != nil {
		return nil, fmt.Errorf("cancelShipment: persist cancellation: %w", err)
	}

	uc.recordShipmentCancelledActivity(ctx, shipment.SalesOrderID, in.UserID, *result)
	return result, nil
}

func (uc *CancelShipmentUseCase) recordShipmentCancelledActivity(ctx context.Context, salesOrderID, userID string, result CancelShipmentResult) {
	if uc.recordActivity == nil {
		return
	}
	_, _ = uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
		SalesOrderID: salesOrderID, UserID: userID, ActivityType: "myparcel_shipment_cancelled",
		Summary:  "Shipment cancelled",
		Metadata: map[string]any{"deleted_on_myparcel": result.DeletedOnMyParcel, "message": result.Message},
	})
}
