// Command repair-stock-review-flags is a one-time reconciliation for
// WooCommerce sales orders left inconsistent by the bug this feature fixes:
// before per-line-item StockStatus/NeedsStockReview/NeedsSKUReview existed,
// UpdateSalesOrderUseCase committed an order's header (status,
// StockDecremented=true) BEFORE attempting its line items' actual stock
// movements -- so an order whose movement then failed with
// ErrInsufficientStock (the "insufficient stock for this batch/ledger"
// errors this fixes) was left claiming StockDecremented=true with no
// stock_movement ever recorded, invisible to staff. Also flags any order
// with a still-unmatched-SKU line item, whether or not stock was involved.
//
// See application/oms.RepairStockReviewFlagsUseCase's own doc comment for
// the exact scan/fix rules. Safe to run more than once -- an order/line
// item already in the correct state is left untouched.
//
// Usage:
//
//	go run ./cmd/repair-stock-review-flags                    # every WooCommerce connection's tenant
//	go run ./cmd/repair-stock-review-flags -connection-id=xyz  # just one connection's tenant
package main

import (
	"context"
	"flag"
	"log"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func main() {
	connectionID := flag.String("connection-id", "", "only repair this connection's tenant (default: every WooCommerce connection's tenant)")
	flag.Parse()

	cfg := config.Load()
	db, err := db.Open(cfg.DBDsn, 5)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	connRepo := repositories.NewConnectionRepo(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	linkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	uc := appoms.NewRepairStockReviewFlagsUseCase(db, salesOrderRepo, lineItemRepo, linkRepo)

	// Same cross-tenant discovery pattern as cmd/backfill-woo-sale-prices:
	// ListByTenant has no WHERE tenant_id of its own, so under bypass_rls
	// this returns every tenant's connections.
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
		if conn.Provider != integrationsDomain.ProviderWooCommerce {
			continue
		}
		if *connectionID != "" && conn.ID != *connectionID {
			continue
		}
		if seenTenants[conn.TenantID] {
			continue
		}
		seenTenants[conn.TenantID] = true
		ran++

		runCtx := tenant.WithTenantID(adminCtx, conn.TenantID)
		result, err := uc.Execute(runCtx)
		if err != nil {
			log.Printf("tenant %s: %v", conn.TenantID, err)
			continue
		}
		log.Printf("tenant %s: %d order(s) scanned, %d order(s) flagged, %d line item(s) fixed",
			conn.TenantID, result.OrdersScanned, result.OrdersFlagged, result.LineItemsFixed)
		for _, id := range result.FlaggedOrderIDs {
			log.Printf("  flagged sales order %s", id)
		}
	}

	if ran == 0 {
		if *connectionID != "" {
			log.Fatalf("no WooCommerce connection found with id %q", *connectionID)
		}
		log.Println("no WooCommerce connections found")
	}
}
