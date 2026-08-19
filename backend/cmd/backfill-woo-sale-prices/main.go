// Command backfill-woo-sale-prices is a one-time reconciliation for
// products imported from WooCommerce before ImportWooProductsUseCase
// mapped Woo's sale_price/on_sale/date_on_sale_to fields at all -- every
// SKU imported before that fix has no sale price recorded regardless of
// whether Woo actually had an active sale configured. Re-scans each Woo
// connection's current catalog and fills in exactly the SKUs still
// missing it (see application/channels.BackfillWooSalePricesUseCase's own
// doc comment for the full gating rules). Safe to run more than once --
// a SKU that already has a sale price recorded is never touched.
//
// Usage:
//
//	go run ./cmd/backfill-woo-sale-prices                    # every WooCommerce connection, every tenant
//	go run ./cmd/backfill-woo-sale-prices -connection-id=xyz  # just one connection
package main

import (
	"context"
	"flag"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func main() {
	connectionID := flag.String("connection-id", "", "only backfill this connection (default: every WooCommerce connection, every tenant)")
	flag.Parse()

	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	connRepo := repositories.NewConnectionRepo(db)
	uc := channelsapp.NewBackfillWooSalePricesUseCase(
		db, connRepo, woocommerce.NewFactory(), cfg.IntegrationsEncryptionSecret,
		repositories.NewProductChannelListingRepo(db), repositories.NewProductRepo(db),
		repositories.NewVariantRepo(db), repositories.NewSKURepo(db),
	)

	// ListByTenant has no WHERE tenant_id clause of its own (see its own
	// doc comment) -- it relies entirely on RLS, so under bypass_rls this
	// returns every tenant's connections, not just one. Same cross-tenant
	// pattern cmd/scheduler's polling loops use.
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

	var ran int
	for _, conn := range conns {
		if conn.Provider != integrationsDomain.ProviderWooCommerce {
			continue
		}
		if *connectionID != "" && conn.ID != *connectionID {
			continue
		}
		ran++

		// Real per-connection identity, not just bypass_rls -- Update
		// writes need the SKU's already-stamped TenantID intact (untouched
		// here), but the connection lookup and every read this use case
		// does should still resolve under this tenant's own context for
		// clarity/consistency with every other cross-tenant command in
		// this codebase (see cmd/scheduler's runCtx pattern).
		runCtx := tenant.WithTenantID(adminCtx, conn.TenantID)
		result, err := uc.Execute(runCtx, channelsapp.BackfillWooSalePricesInput{ConnectionID: conn.ID})
		if err != nil {
			log.Printf("connection %s (%s): %v", conn.ID, conn.StoreURL, err)
			continue
		}
		log.Printf("connection %s (%s): %d candidate(s), %d updated, %d error(s)",
			conn.ID, conn.StoreURL, result.Candidates, result.Updated, len(result.Errors))
		for _, e := range result.Errors {
			log.Printf("  %s", e)
		}
	}

	if ran == 0 {
		if *connectionID != "" {
			log.Fatalf("no WooCommerce connection found with id %q", *connectionID)
		}
		log.Println("no WooCommerce connections found")
	}
}
