package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// checkBolShipmentConfirmationsLimit bounds one run -- the job runs every
// few minutes and only ever has the last few minutes' shipments pending.
const checkBolShipmentConfirmationsLimit = 200

// CheckBolShipmentConfirmationsUseCase is the check_bol_shipment_confirmations
// global job: for every pending BolShipmentConfirmation (any tenant), ask
// Bol how processing went and store the final answer, so a shipment Bol
// refused shows up on the order page. A confirmation Bol is still
// processing stays pending for the next run. One failed lookup is logged
// and skipped; only failing to list the pending ones fails the run.
type CheckBolShipmentConfirmationsUseCase struct {
	db            *gorm.DB
	confirmations channelsDomain.BolShipmentConfirmationRepository
	clients       bolClientSource
	outbox        eventrecord.Enqueuer
	now           func() time.Time
}

func NewCheckBolShipmentConfirmationsUseCase(
	db *gorm.DB,
	confirmations channelsDomain.BolShipmentConfirmationRepository,
	clients bolClientSource,
	outbox eventrecord.Enqueuer,
	now func() time.Time,
) *CheckBolShipmentConfirmationsUseCase {
	return &CheckBolShipmentConfirmationsUseCase{db: db, confirmations: confirmations, clients: clients, outbox: outbox, now: now}
}

func (uc *CheckBolShipmentConfirmationsUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []channelsDomain.BolShipmentConfirmation
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		pending, err = uc.confirmations.ListPending(ctx, checkBolShipmentConfirmationsLimit)
		return err
	})
	if err != nil {
		return fmt.Errorf("checkBolShipmentConfirmations: %w", err)
	}

	clients := map[string]integrationsDomain.BolClient{}
	for _, c := range pending {
		tenantCtx := tenant.WithTenantID(adminCtx, c.TenantID)
		client, ok := clients[c.ConnectionID]
		if !ok {
			if client, err = uc.clients.ClientFor(tenantCtx, c.ConnectionID); err != nil {
				log.Printf("checkBolShipmentConfirmations: confirmation %s: %v", c.ID, err)
				continue
			}
			clients[c.ConnectionID] = client
		}
		if err := uc.check(tenantCtx, client, c); err != nil {
			log.Printf("checkBolShipmentConfirmations: confirmation %s: %v", c.ID, err)
		}
	}
	return nil
}

func (uc *CheckBolShipmentConfirmationsUseCase) check(ctx context.Context, client integrationsDomain.BolClient, c channelsDomain.BolShipmentConfirmation) error {
	report, err := client.GetProcessStatus(ctx, c.ProcessStatusID)
	if err != nil {
		return err
	}
	status, err := channelsDomain.ParseBolConfirmationStatus(report.Status)
	if err != nil {
		return err
	}
	if status == channelsDomain.BolConfirmationPending {
		return nil
	}
	if err := c.Resolve(status, report.ErrorMessage, uc.now()); err != nil {
		return err
	}
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.confirmations.SaveResolution(ctx, &c); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BolShipmentConfirmationResolved, c.ID,
			eventrecord.NewChanges().Set("status", string(channelsDomain.BolConfirmationPending), string(c.Status)),
			eventrecord.Data{
				"sales_order_id": c.SalesOrderID, "external_order_id": c.ExternalOrderID,
				"process_status_id": c.ProcessStatusID, "error_message": c.ErrorMessage,
			})
	})
}
