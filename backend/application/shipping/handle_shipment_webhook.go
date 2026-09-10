package shipping

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// myParcelStatusChangeHook is one entry of MyParcel's shipment_status_change
// webhook payload -- `{"data":{"hooks":[{...}]}}`. Only the fields this
// codebase actually uses are modeled; MyParcel's own numeric `status` here
// isn't paired with a `status_main` string in the webhook payload itself
// (unlike the tracktrace GET response), so HandleShipmentWebhookUseCase
// treats the webhook as a "something changed, go check" trigger and calls
// GetTrackTrace for the authoritative current status rather than
// interpreting the bare numeric code -- one extra API call per event, but
// avoids guessing a numeric-status meaning that was never confirmed
// against real docs.
type myParcelStatusChangeHook struct {
	ShipmentID int64  `json:"shipment_id"`
	Status     int    `json:"status"`
	Barcode    string `json:"barcode"`
}

type myParcelWebhookPayload struct {
	Data struct {
		Hooks []myParcelStatusChangeHook `json:"hooks"`
	} `json:"data"`
}

// HandleShipmentWebhookUseCase is the MyParcel branch of the webhook cold
// path's per-topic dispatch (see application/webhooks.ProcessWebhookEventUseCase,
// which calls this after its own persist+dedupe step -- that generic
// "prove this event was received" behavior is unchanged for every other
// provider). For each shipment_id in the payload: look up the local
// Shipment, fetch its authoritative current status from MyParcel, and
// record it via applyStatusUpdate -- which (this use case always supplies a
// real outboxRepo) raises a ShipmentFulfillmentSignal event in the same
// transaction. What that means for the SalesOrder is entirely
// ApplyShipmentFulfillmentSignalUseCase's decision (see its doc), not this
// use case's -- it only reports what MyParcel said.
type HandleShipmentWebhookUseCase struct {
	db              *gorm.DB
	shipmentRepo    shippingDomain.ShipmentRepository
	statusEventRepo shippingDomain.ShipmentStatusEventRepository
	connRepo        integrationsDomain.ConnectionRepository
	salesOrderRepo  omsDomain.SalesOrderRepository
	carrierFactory  integrationsDomain.CarrierClientFactory
	encSecret       string
	outboxRepo      outbox.Repository
}

func NewHandleShipmentWebhookUseCase(
	db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	connRepo integrationsDomain.ConnectionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	carrierFactory integrationsDomain.CarrierClientFactory,
	encryptionSecret string,
	outboxRepo outbox.Repository,
) *HandleShipmentWebhookUseCase {
	return &HandleShipmentWebhookUseCase{
		db: db, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo, connRepo: connRepo,
		salesOrderRepo: salesOrderRepo, carrierFactory: carrierFactory, encSecret: encryptionSecret,
		outboxRepo: outboxRepo,
	}
}

func (uc *HandleShipmentWebhookUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	var payload myParcelWebhookPayload
	if err := json.Unmarshal(envelope.RawPayload, &payload); err != nil {
		return fmt.Errorf("handleShipmentWebhook: decode payload: %w", err)
	}

	ctx = tenant.WithTenantID(ctx, envelope.TenantID)
	for _, hook := range payload.Data.Hooks {
		if err := uc.handleOne(ctx, hook); err != nil {
			return fmt.Errorf("handleShipmentWebhook: shipment %d: %w", hook.ShipmentID, err)
		}
	}
	return nil
}

func (uc *HandleShipmentWebhookUseCase) handleOne(ctx context.Context, hook myParcelStatusChangeHook) error {
	var shipment shippingDomain.Shipment
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.shipmentRepo.FindByExternalShipmentID(ctx, strconv.FormatInt(hook.ShipmentID, 10))
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
		// A shipment_id this tenant doesn't know about (never created here,
		// or a stale webhook from a since-deleted shipment) -- nothing to
		// update, not a fault worth nak-ing/retrying forever.
		return nil
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt api key: %w", err)
	}
	client, err := uc.carrierFactory.NewClient(apiKey)
	if err != nil {
		return fmt.Errorf("build client: %w", err)
	}

	tt, err := client.GetTrackTrace(ctx, strconv.FormatInt(hook.ShipmentID, 10))
	if err != nil {
		return fmt.Errorf("get tracktrace: %w", err)
	}

	_, err = applyStatusUpdate(ctx, uc.db, uc.shipmentRepo, uc.statusEventRepo, uc.salesOrderRepo,
		shipment.ID, shipment.SalesOrderID, tt, shippingDomain.StatusEventSourceWebhook, hook.Barcode, uc.outboxRepo)
	return err
}
