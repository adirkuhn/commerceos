package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ReconcileWindow is the default "no update in this long" threshold
// ReconcileShipmentsUseCase re-polls against -- mirrors the reasoning of
// channels.SyncEntityOrdersReconciliation's reconciliationWindow (self-heal
// whatever a missed/never-received webhook left stale), just shorter since
// a shipment's status changes on the order of days, not weeks.
const ReconcileWindow = 24 * time.Hour

// reconcileBatchLimit bounds one run's work -- this runs on a scheduler
// tick, not a hot path, but an unbounded scan across every stale shipment
// for a busy tenant would still be an unnecessarily large single poll.
const reconcileBatchLimit = 200

type ReconcileShipmentsResult struct {
	Checked int
	Updated int
	Errors  []string
}

// ReconcileShipmentsUseCase is the "global scheduler to process old orders
// without webhook notifications/missed notifications" the design called
// for -- re-polls MyParcel's own tracktrace for any non-terminal Shipment
// on this connection that hasn't seen a status update inside ReconcileWindow,
// self-healing whatever HandleShipmentWebhookUseCase's webhook path ever
// missed. A per-shipment failure is appended to Errors and the run
// continues -- one bad shipment must not block the rest, same
// partial-failure-tolerant convention as channels' own scheduled runners
// (see cmd/scheduler/dispatch.go's resultErr).
type ReconcileShipmentsUseCase struct {
	db              *gorm.DB
	shipmentRepo    shippingDomain.ShipmentRepository
	statusEventRepo shippingDomain.ShipmentStatusEventRepository
	connRepo        integrationsDomain.ConnectionRepository
	salesOrderRepo  omsDomain.SalesOrderRepository
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
}

func NewReconcileShipmentsUseCase(
	db *gorm.DB,
	shipmentRepo shippingDomain.ShipmentRepository,
	statusEventRepo shippingDomain.ShipmentStatusEventRepository,
	connRepo integrationsDomain.ConnectionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
) *ReconcileShipmentsUseCase {
	return &ReconcileShipmentsUseCase{
		db: db, shipmentRepo: shipmentRepo, statusEventRepo: statusEventRepo, connRepo: connRepo,
		salesOrderRepo: salesOrderRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret,
	}
}

func (uc *ReconcileShipmentsUseCase) Execute(ctx context.Context, connectionID string) (*ReconcileShipmentsResult, error) {
	var conn integrationsDomain.Connection
	var stale []shippingDomain.Shipment
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *c

		all, err := uc.shipmentRepo.FindStaleNonTerminal(ctx, time.Now().Add(-ReconcileWindow), reconcileBatchLimit)
		if err != nil {
			return err
		}
		for _, s := range all {
			if s.ConnectionID == connectionID {
				stale = append(stale, s)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reconcileShipments: %w", err)
	}

	result := &ReconcileShipmentsResult{}
	if len(stale) == 0 {
		return result, nil
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("reconcileShipments: decrypt api key: %w", err)
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("reconcileShipments: build client: %w", err)
	}

	for _, s := range stale {
		result.Checked++
		tt, err := client.GetTrackTrace(ctx, s.MyParcelShipmentID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("shipment %s: get tracktrace: %v", s.ID, err))
			continue
		}
		if tt.StatusMain == s.StatusMain && tt.StatusCode == s.Status {
			continue // nothing actually changed -- still touch nothing, just confirms it's genuinely stalled, not missed
		}
		if err := applyStatusUpdate(ctx, uc.db, uc.shipmentRepo, uc.statusEventRepo, uc.salesOrderRepo,
			s.ID, s.SalesOrderID, tt, shippingDomain.StatusEventSourceReconciliation); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("shipment %s: apply update: %v", s.ID, err))
			continue
		}
		result.Updated++
	}
	return result, nil
}
