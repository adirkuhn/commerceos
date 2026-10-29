// Command repair-unmanaged-woo-stock is a one-time repair for Woo products
// created without manage_stock although their CommerceOS SKU is physical
// -- what happened to every variant added to an already-listed product
// (e.g. "split into packages") before SyncProductToChannelsUseCase sent
// stock on create. Woo sells such a product as in stock with no limit.
// Force-syncs stock for each one it finds (see
// application/channels.RepairUnmanagedWooStockUseCase). Safe to run more
// than once. Variation listings aren't checked (counted in the output).
//
// Usage:
//
//	go run ./cmd/repair-unmanaged-woo-stock -dry-run                   # list only, every WooCommerce connection
//	go run ./cmd/repair-unmanaged-woo-stock                            # repair, every WooCommerce connection
//	go run ./cmd/repair-unmanaged-woo-stock -connection-id=xyz         # just one connection
package main

import (
	"context"
	"flag"
	"log"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func main() {
	connectionID := flag.String("connection-id", "", "only repair this connection (default: every WooCommerce connection, every tenant)")
	dryRun := flag.Bool("dry-run", false, "list unmanaged Woo products without pushing stock")
	flag.Parse()

	cfg := config.Load()
	db, err := db.Open(cfg.DBDsn, 5)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	connRepo := repositories.NewConnectionRepo(db)
	wooFactory := woocommerce.NewFactory()
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)
	outbox := repositories.NewOutboxRepository(db)
	stockLevelUC := invapp.NewGetSKUStockLevelUseCase(
		db, repositories.NewStockMovementRepo(db), repositories.NewBatchRepo(db), repositories.NewTenantSettingsRepo(db),
	)
	// No Bol pusher: this repair is about Woo listings only, and
	// SyncStockToChannelsUseCase skips Bol when it's nil.
	syncStockUC := channelsapp.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, repositories.NewSyncLogRepo(db), connRepo, wooFactory, stockLevelUC,
		cfg.IntegrationsEncryptionSecret, nil, outbox,
	)
	uc := channelsapp.NewRepairUnmanagedWooStockUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		listingRepo, repositories.NewProductRepo(db), variantRepo, skuRepo, syncStockUC,
	)

	// ListByTenant relies on RLS alone -- under bypass_rls it returns every
	// tenant's connections (same pattern as cmd/backfill-woo-sale-prices).
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

		runCtx := tenant.AsTenant(adminCtx, conn.TenantID)
		result, err := uc.Execute(runCtx, channelsapp.RepairUnmanagedWooStockInput{ConnectionID: conn.ID, DryRun: *dryRun})
		if err != nil {
			log.Printf("connection %s (%s): %v", conn.ID, conn.StoreURL, err)
			continue
		}
		log.Printf("connection %s (%s): %d checked, %d unmanaged, %d repaired, %d variation listing(s) not checked, %d error(s)",
			conn.ID, conn.StoreURL, result.Checked, len(result.Unmanaged), result.Repaired, result.SkippedVariations, len(result.Errors))
		for _, u := range result.Unmanaged {
			log.Printf("  unmanaged: woo #%s %q (sku %s)", u.ExternalID, u.Name, u.SKUID)
		}
		for _, e := range result.Errors {
			log.Printf("  error: %s", e)
		}
	}

	if ran == 0 {
		if *connectionID != "" {
			log.Fatalf("no WooCommerce connection found with id %q", *connectionID)
		}
		log.Println("no WooCommerce connections found")
	}
	if *dryRun {
		log.Println("dry run: nothing pushed")
	}
}
