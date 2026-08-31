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

// eventDispatchStats is one webhook_events row's per-hook outcome tally --
// shared shape both BackfillShipmentWebhookDispatchUseCase (aggregates
// across many rows) and RetryWebhookEventDispatchUseCase (just reports on
// its one row) read from shipmentWebhookDispatcher.processEvent.
type eventDispatchStats struct {
	applied, duplicate, unknownShipment int
	failed                              bool
}

// shipmentWebhookDispatcher is the per-shipment resolution and
// applyStatusUpdate path HandleShipmentWebhookUseCase/ReconcileShipments-
// UseCase already share, reused a third time here for reprocessing a
// webhook_events row after the fact -- BackfillShipmentWebhookDispatchUseCase
// (many old rows, one run) and RetryWebhookEventDispatchUseCase (one row,
// on demand) both just wrap this with their own entry point, so neither
// can ever produce a different outcome than live processing would have.
type shipmentWebhookDispatcher struct {
	db               *gorm.DB
	webhookEventRepo webhooksDomain.WebhookEventRepository
	shipmentRepo     shippingDomain.ShipmentRepository
	statusEventRepo  shippingDomain.ShipmentStatusEventRepository
	connRepo         integrationsDomain.ConnectionRepository
	salesOrderRepo   omsDomain.SalesOrderRepository
	myParcelFactory  integrationsDomain.MyParcelClientFactory
	encSecret        string
	// updateSalesOrder may be nil (a deployment that hasn't wired it, or a
	// caller like BackfillShipmentWebhookDispatchUseCase's cmd binary that
	// deliberately doesn't) -- completeOrderOnDelivery is then a no-op, same
	// "optional dependency" convention as ProcessWebhookEventUseCase's
	// shipmentHandler.
	updateSalesOrder salesOrderStatusUpdater
}

// processEvent parses one webhook_events row's payload and resolves each
// hook via cache (pass a fresh map for a single-event caller, or share one
// across a batch to dedupe repeated shipments -- see
// shipmentDispatchOutcome's doc comment), then marks the row's dispatch
// result. The returned error is only ever a failure to persist that mark
// itself (Processed path) -- a dispatch failure is recorded in the
// returned stats/the row's own dispatch_error, not returned as an error,
// since that's an expected, retryable outcome, not a fault in this method.
func (d *shipmentWebhookDispatcher) processEvent(ctx context.Context, event webhooksDomain.WebhookEvent, cache map[int64]*shipmentDispatchOutcome) (eventDispatchStats, error) {
	var stats eventDispatchStats

	var payload myParcelWebhookPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		_ = d.markDispatchResult(ctx, event.ID, webhooksDomain.WebhookEventStatusFailed, fmt.Sprintf("decode payload: %v", err))
		stats.failed = true
		return stats, nil
	}

	eventErr := ""
	for _, hook := range payload.Data.Hooks {
		outcome, alreadySeenThisRun := cache[hook.ShipmentID]
		if !alreadySeenThisRun {
			outcome = d.resolveShipment(ctx, hook.ShipmentID)
			cache[hook.ShipmentID] = outcome
		}
		switch {
		case outcome.err != nil:
			eventErr = outcome.err.Error()
		case outcome.skippedUnknownShipment:
			stats.unknownShipment++
		// A cache hit means an earlier hook in this same call already wrote
		// (or determined there was nothing to write for) this shipment --
		// this one causes no new write of its own, so it always counts as
		// a duplicate, even if the first occurrence was Applied.
		case !alreadySeenThisRun && outcome.applied:
			stats.applied++
		default:
			stats.duplicate++
		}
	}

	if eventErr != "" {
		_ = d.markDispatchResult(ctx, event.ID, webhooksDomain.WebhookEventStatusFailed, eventErr)
		stats.failed = true
		return stats, nil
	}
	if err := d.markDispatchResult(ctx, event.ID, webhooksDomain.WebhookEventStatusProcessed, ""); err != nil {
		return stats, fmt.Errorf("mark processed: %w", err)
	}
	return stats, nil
}

func (d *shipmentWebhookDispatcher) resolveShipment(ctx context.Context, myParcelShipmentID int64) *shipmentDispatchOutcome {
	var shipment shippingDomain.Shipment
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, d.db, func(ctx context.Context) error {
		s, err := d.shipmentRepo.FindByMyParcelShipmentID(ctx, myParcelShipmentID)
		if err != nil {
			return err
		}
		shipment = *s
		c, err := d.connRepo.FindByID(ctx, s.ConnectionID)
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

	apiKey, err := crypto.Decrypt(d.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return &shipmentDispatchOutcome{err: fmt.Errorf("decrypt api key: %w", err)}
	}
	client, err := d.myParcelFactory.NewClient(apiKey)
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
	if _, err := applyStatusUpdate(ctx, d.db, d.shipmentRepo, d.statusEventRepo, d.salesOrderRepo,
		shipment.ID, shipment.SalesOrderID, tt, shippingDomain.StatusEventSourceBackfill); err != nil {
		return &shipmentDispatchOutcome{err: fmt.Errorf("apply status update: %w", err)}
	}
	if err := completeOrderOnDelivery(ctx, d.updateSalesOrder, tt.StatusMain, shipment.SalesOrderID); err != nil {
		return &shipmentDispatchOutcome{err: err}
	}
	return &shipmentDispatchOutcome{applied: true}
}

func (d *shipmentWebhookDispatcher) markDispatchResult(ctx context.Context, eventID string, status webhooksDomain.WebhookEventStatus, dispatchError string) error {
	return tenant.WithTenant(ctx, d.db, func(ctx context.Context) error {
		return d.webhookEventRepo.MarkDispatchResult(ctx, eventID, status, dispatchError)
	})
}

// BackfillShipmentWebhookDispatchUseCase is a one-time reprocessing of
// myparcel shipment_status_change webhooks received before dispatch-status
// tracking existed (see application/webhooks.ProcessWebhookEventUseCase),
// so their outcome becomes visible on the Messages tab instead of staying
// "received" forever.
type BackfillShipmentWebhookDispatchUseCase struct {
	dispatcher *shipmentWebhookDispatcher
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
	updateSalesOrder salesOrderStatusUpdater,
) *BackfillShipmentWebhookDispatchUseCase {
	return &BackfillShipmentWebhookDispatchUseCase{dispatcher: &shipmentWebhookDispatcher{
		db: db, webhookEventRepo: webhookEventRepo, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo,
		connRepo: connRepo, salesOrderRepo: salesOrderRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
		updateSalesOrder: updateSalesOrder,
	}}
}

func (uc *BackfillShipmentWebhookDispatchUseCase) Execute(ctx context.Context) (*BackfillShipmentWebhookDispatchResult, error) {
	var events []webhooksDomain.WebhookEvent
	err := tenant.WithTenant(ctx, uc.dispatcher.db, func(ctx context.Context) error {
		var err error
		events, err = uc.dispatcher.webhookEventRepo.List(ctx, "myparcel", backfillFetchLimit)
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

		stats, err := uc.dispatcher.processEvent(ctx, event, cache)
		if err != nil {
			return nil, fmt.Errorf("backfillShipmentWebhookDispatch: %w", err)
		}
		result.Applied += stats.applied
		result.SkippedDuplicate += stats.duplicate
		result.SkippedUnknownShipment += stats.unknownShipment
		if stats.failed {
			result.Failed++
		}
	}
	return result, nil
}
