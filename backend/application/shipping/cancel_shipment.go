package shipping

import (
	"context"
	"fmt"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
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
// CancelShipmentResult: DeletedOnMyParcel and CarrierError (MyParcel's own
// wording when it rejected the delete) let clients show the outcome in the
// reader's language; Message is the English text.
type CancelShipmentResult struct {
	DeletedOnMyParcel bool
	CarrierError      string
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
	outbox         eventrecord.Enqueuer
}

func NewCancelShipmentUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	carrierFactory integrationsDomain.CarrierClientFactory,
	encryptionSecret string,
	recordActivity recordActivityUC,
	outbox eventrecord.Enqueuer,
) *CancelShipmentUseCase {
	return &CancelShipmentUseCase{
		db: db, connRepo: connRepo, shipmentRepo: shipmentRepo,
		carrierFactory: carrierFactory, encSecret: encryptionSecret,
		recordActivity: recordActivity, now: time.Now, outbox: outbox,
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

	client, err := buildCarrierClient(uc.carrierFactory, uc.encSecret, conn)
	if err != nil {
		return nil, fmt.Errorf("cancelShipment: %w", err)
	}

	deleteErr := client.CancelShipment(ctx, shipment.ExternalShipmentID)
	outcome := shippingDomain.CancelOutcome{DeletedOnCarrier: deleteErr == nil}
	if deleteErr != nil {
		outcome.CarrierError = deleteErr.Error()
	}
	result := &CancelShipmentResult{DeletedOnMyParcel: outcome.DeletedOnCarrier, CarrierError: outcome.CarrierError, Message: outcome.Note()}

	cancelledAt := uc.now()
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.shipmentRepo.MarkCancelled(ctx, shipment.ID, cancelledAt, outcome); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ShipmentCancelled, shipment.ID, eventrecord.Data{
			"sales_order_id": shipment.SalesOrderID, "deleted_on_carrier": result.DeletedOnMyParcel, "note": result.Message,
		})
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
	if _, err := uc.recordActivity.Execute(ctx, appoms.RecordSalesOrderActivityInput{
		SalesOrderID: salesOrderID, UserID: userID, ActivityType: omsDomain.ActivityTypeShipmentCancelled,
		Summary:  "Shipment cancelled",
		Metadata: map[string]any{"deleted_on_myparcel": result.DeletedOnMyParcel, "message": result.Message},
	}); err != nil {
		log.Printf("cancelShipment: record activity for %s: %v", salesOrderID, err)
	}
}
