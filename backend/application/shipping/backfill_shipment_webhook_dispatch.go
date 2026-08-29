package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// backfillFetchLimit bounds one run's scan of webhook_events -- this is a
// one-time reprocessing for the (small, per the approved scope) backlog of
// webhooks received before dispatch-status tracking existed, not a
// recurring job, so a single generous batch is enough; rerun if a tenant
// ever has more than this many.
const backfillFetchLimit = 1000

type BackfillShipmentWebhookDispatchResult struct {
	// Checked is every myparcel shipment_status_change webhook_events row
	// still at its default Received status (never dispatched -- received
	// before this tracking existed).
	Checked int
	// Applied is a hook that found its shipment, fetched a status different
	// from what's already recorded, and wrote a new ShipmentStatusEvent
	// (source=Backfill) -- linking to and, when it's the shipped
	// transition, marking the SalesOrder fulfilled, same as a live webhook.
	Applied int
	// SkippedDuplicate is a hook whose shipment's current status already
	// matches what MyParcel reports right now -- no new event written, the
	// webhook_events row is still marked Processed (the outcome IS known,
	// it just wasn't a change).
	SkippedDuplicate int
	// SkippedUnknownShipment is a hook for a shipment_id this tenant has no
	// local Shipment for (same as the live webhook path) -- nothing to do.
	SkippedUnknownShipment int
	// Failed is a webhook_events row marked Failed (a real error resolving
	// its shipment/connection/provider call), not a normal skip.
	Failed int
}

// shipmentDispatchOutcome memoizes one MyParcel shipment_id's resolution
// for the lifetime of a single Execute call -- several old webhook_events
// rows commonly reference the same shipment (see the result fields' own
// doc comments), and MyParcel's tracktrace only ever answers "what's the
// status right now," so re-fetching it per row would be wasteful and could
// re-record the exact same "current status" as a second, redundant
// timeline entry for the same underlying change.
type shipmentDispatchOutcome struct {
	skippedUnknownShipment bool
	applied                bool
	err                    error
}

// BackfillShipmentWebhookDispatchUseCase is a one-time reprocessing of
// myparcel shipment_status_change webhooks received before dispatch-status
// tracking existed (see application/webhooks.ProcessWebhookEventUseCase),
// so their outcome becomes visible on the Messages tab instead of staying
// "received" forever. Reuses the exact same per-shipment resolution and
// applyStatusUpdate path HandleShipmentWebhookUseCase/ReconcileShipments-
// UseCase already share -- including ReconcileShipmentsUseCase's own
// "nothing actually changed" skip -- so a backfill run can never produce a
// different outcome than what live processing would have.
type BackfillShipmentWebhookDispatchUseCase struct {
	db               *gorm.DB
	webhookEventRepo webhooksDomain.WebhookEventRepository
	shipmentRepo     shippingDomain.ShipmentRepository
	statusEventRepo  shippingDomain.ShipmentStatusEventRepository
	connRepo         integrationsDomain.ConnectionRepository
	salesOrderRepo   omsDomain.SalesOrderRepository
	myParcelFactory  integrationsDomain.MyParcelClientFactory
	encSecret        string
}

func NewBackfillShipmentWebhookDispatchUseCase(
	db *gorm.DB,
	webhookEventRepo webhooksDomain.WebhookEventRepository,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	connRepo integrationsDomain.ConnectionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
) *BackfillShipmentWebhookDispatchUseCase {
	return &BackfillShipmentWebhookDispatchUseCase{
		db: db, webhookEventRepo: webhookEventRepo, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo,
		connRepo: connRepo, salesOrderRepo: salesOrderRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
	}
}

func (uc *BackfillShipmentWebhookDispatchUseCase) Execute(ctx context.Context) (*BackfillShipmentWebhookDispatchResult, error) {
	var events []webhooksDomain.WebhookEvent
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		events, err = uc.webhookEventRepo.List(ctx, "myparcel", backfillFetchLimit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("backfillShipmentWebhookDispatch: list events: %w", err)
	}

	result := &BackfillShipmentWebhookDispatchResult{}
	cache := map[int64]*shipmentDispatchOutcome{}

	for _, event := range events {
		if event.Topic != "shipment_status_change" || event.Status != webhooksDomain.WebhookEventStatusReceived {
			continue
		}
		result.Checked++

		var payload myParcelWebhookPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			_ = uc.markDispatchResult(ctx, event.ID, webhooksDomain.WebhookEventStatusFailed, fmt.Sprintf("decode payload: %v", err))
			result.Failed++
			continue
		}

		eventErr := ""
		for _, hook := range payload.Data.Hooks {
			outcome, alreadySeenThisRun := cache[hook.ShipmentID]
			if !alreadySeenThisRun {
				outcome = uc.resolveShipment(ctx, hook.ShipmentID)
				cache[hook.ShipmentID] = outcome
			}
			switch {
			case outcome.err != nil:
				eventErr = outcome.err.Error()
			case outcome.skippedUnknownShipment:
				result.SkippedUnknownShipment++
			// A cache hit means an earlier row in this same run already
			// wrote (or determined there was nothing to write for) this
			// shipment -- this row causes no new write of its own, so it
			// always counts as a duplicate from here, even if the first
			// occurrence was Applied.
			case !alreadySeenThisRun && outcome.applied:
				result.Applied++
			default:
				result.SkippedDuplicate++
			}
		}

		if eventErr != "" {
			_ = uc.markDispatchResult(ctx, event.ID, webhooksDomain.WebhookEventStatusFailed, eventErr)
			result.Failed++
			continue
		}
		if err := uc.markDispatchResult(ctx, event.ID, webhooksDomain.WebhookEventStatusProcessed, ""); err != nil {
			return nil, fmt.Errorf("backfillShipmentWebhookDispatch: mark processed: %w", err)
		}
	}
	return result, nil
}

func (uc *BackfillShipmentWebhookDispatchUseCase) resolveShipment(ctx context.Context, myParcelShipmentID int64) *shipmentDispatchOutcome {
	var shipment shippingDomain.Shipment
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.shipmentRepo.FindByMyParcelShipmentID(ctx, myParcelShipmentID)
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
		if errors.Is(err, shippingDomain.ErrShipmentNotFound) {
			return &shipmentDispatchOutcome{skippedUnknownShipment: true}
		}
		return &shipmentDispatchOutcome{err: err}
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return &shipmentDispatchOutcome{err: fmt.Errorf("decrypt api key: %w", err)}
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return &shipmentDispatchOutcome{err: fmt.Errorf("build client: %w", err)}
	}
	tt, err := client.GetTrackTrace(ctx, myParcelShipmentID)
	if err != nil {
		return &shipmentDispatchOutcome{err: fmt.Errorf("get tracktrace: %w", err)}
	}
	if tt.StatusMain == shipment.StatusMain && tt.StatusCode == shipment.Status {
		return &shipmentDispatchOutcome{applied: false}
	}
	if _, err := applyStatusUpdate(ctx, uc.db, uc.shipmentRepo, uc.statusEventRepo, uc.salesOrderRepo,
		shipment.ID, shipment.SalesOrderID, tt, shippingDomain.StatusEventSourceBackfill); err != nil {
		return &shipmentDispatchOutcome{err: fmt.Errorf("apply status update: %w", err)}
	}
	return &shipmentDispatchOutcome{applied: true}
}

func (uc *BackfillShipmentWebhookDispatchUseCase) markDispatchResult(ctx context.Context, eventID string, status webhooksDomain.WebhookEventStatus, dispatchError string) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.webhookEventRepo.MarkDispatchResult(ctx, eventID, status, dispatchError)
	})
}
