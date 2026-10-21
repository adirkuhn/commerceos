// Command backfill-webhook-dispatch is a one-time reprocessing of myparcel
// shipment_status_change webhooks received before webhook_events gained
// its status column (see application/webhooks.ProcessWebhookEventUseCase)
// -- every such row stays "received" forever otherwise, since nothing
// retroactively knows whether the old binary's dispatch actually
// succeeded. Re-fetches each affected shipment's current status from
// MyParcel (once per shipment, however many old rows reference it) and
// applies it through the exact same path a live webhook would, so a
// tenant that was already fully up to date ends up with every row simply
// marked "processed" and no new timeline entries, while one that missed a
// real transition gets it applied now, order fulfillment included.
// Safe to run more than once -- an already-processed/failed row is never
// re-touched, and re-fetching an unchanged status is a no-op.
//
// Usage:
//
//	go run ./cmd/backfill-webhook-dispatch                    # every tenant with a MyParcel connection
//	go run ./cmd/backfill-webhook-dispatch -tenant-id=xyz      # just one tenant
package main

import (
	"context"
	"flag"
	"log"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/carriers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func main() {
	tenantID := flag.String("tenant-id", "", "only backfill this tenant (default: every tenant with a MyParcel connection)")
	flag.Parse()

	cfg := config.Load()
	db, err := db.Open(cfg.DBDsn, 5)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	connRepo := repositories.NewConnectionRepo(db)
	// updateSalesOrder/whatsappNotifier are both nil here -- this is a
	// one-time reprocessing of the pre-dispatch-tracking backlog (see this
	// file's own doc comment), out of scope for wiring the full OMS/
	// inventory/WhatsApp stack just to complete an order or send a
	// notification on a "delivered" shipment discovered mid-backfill; both
	// paths stay a no-op for this tool (see completeOrderOnDelivery/
	// notifyWhatsAppOnDelivery's docs).
	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(
		db, repositories.NewWebhookEventRepo(db), repositories.NewShipmentRepo(db), repositories.NewShipmentStatusEventRepo(db),
		connRepo, repositories.NewSalesOrderRepo(db), carriers.NewDefaultRegistry(), cfg.IntegrationsEncryptionSecret, nil, nil,
	)

	// ListByTenant has no WHERE tenant_id clause of its own -- under
	// bypass_rls this returns every tenant's connections, not just one.
	// Same cross-tenant pattern cmd/backfill-woo-sale-prices and
	// cmd/scheduler's polling loops use.
	adminCtx := tenant.WithSuperAdmin(context.Background())
	var conns []integrationsDomain.Connection
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		conns, err = connRepo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		log.Fatalf("list connections: %v", err)
	}

	seenTenants := map[string]bool{}
	var ran int
	for _, conn := range conns {
		if conn.Provider != integrationsDomain.ProviderMyParcel {
			continue
		}
		if *tenantID != "" && conn.TenantID != *tenantID {
			continue
		}
		if seenTenants[conn.TenantID] {
			continue // one MyParcel connection is enough to run this tenant's backfill once
		}
		seenTenants[conn.TenantID] = true
		ran++

		runCtx := tenant.WithTenantID(adminCtx, conn.TenantID)
		result, err := uc.Execute(runCtx)
		if err != nil {
			log.Printf("tenant %s: %v", conn.TenantID, err)
			continue
		}
		log.Printf("tenant %s: %d checked, %d applied, %d duplicate, %d unknown shipment, %d failed",
			conn.TenantID, result.Checked, result.Applied, result.SkippedDuplicate, result.SkippedUnknownShipment, result.Failed)
	}

	if ran == 0 {
		if *tenantID != "" {
			log.Fatalf("no MyParcel connection found for tenant %q", *tenantID)
		}
		log.Println("no MyParcel connections found")
	}
}
