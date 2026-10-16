// Command repair-partially-received-pos is a one-time reconciliation for
// purchase orders left partially_received by the old booking finish, which
// judged a PO by the finishing booking's scans only: a PO delivered in
// parts whose later delivery brought the rest never became received.
//
// See application/procurement.ReconcilePartiallyReceivedPurchaseOrdersUseCase
// for the rules. Dry run by default -- it lists what it would change; pass
// -dry-run=false to apply. Safe to run more than once.
//
// Usage:
//
//	/repair-partially-received-pos                          # dry run, every tenant
//	/repair-partially-received-pos -dry-run=false           # apply, every tenant
//	/repair-partially-received-pos -tenant-id=<uuid> ...    # one tenant only
package main

import (
	"context"
	"flag"
	"log"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "only list what would change (pass -dry-run=false to apply)")
	tenantID := flag.String("tenant-id", "", "only repair this tenant (default: every tenant)")
	flag.Parse()

	cfg := config.Load()
	db, err := db.Open(cfg.DBDsn, 5)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	poRepo := repositories.NewPurchaseOrderRepo(db)
	uc := appprocurement.NewReconcilePartiallyReceivedPurchaseOrdersUseCase(db, poRepo, repositories.NewBookingRepo(db),
		repositories.NewPurchaseOrderLineItemRepo(db), repositories.NewBookingScanRepo(db), repositories.NewOutboxRepository(db))

	// Under bypass_rls List has no tenant filter: every tenant's POs. Only
	// tenants with a partially_received PO need a run.
	adminCtx := tenant.WithSuperAdmin(context.Background())
	var tenants []string
	seen := map[string]bool{}
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		pos, err := poRepo.List(ctx)
		if err != nil {
			return err
		}
		for _, po := range pos {
			if po.Status != procurementDomain.PurchaseOrderStatusPartiallyReceived || seen[po.TenantID] {
				continue
			}
			if *tenantID != "" && po.TenantID != *tenantID {
				continue
			}
			seen[po.TenantID] = true
			tenants = append(tenants, po.TenantID)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("list purchase orders: %v", err)
	}
	if len(tenants) == 0 {
		log.Println("no partially_received purchase orders found")
		return
	}

	mode := "APPLY"
	if *dryRun {
		mode = "DRY RUN (nothing written; pass -dry-run=false to apply)"
	}
	log.Printf("%s: %d tenant(s) with partially_received purchase orders", mode, len(tenants))

	var failed bool
	for _, id := range tenants {
		result, err := uc.Execute(tenant.AsTenant(adminCtx, id), appprocurement.ReconcilePartiallyReceivedInput{DryRun: *dryRun})
		if result != nil {
			log.Printf("tenant %s: %d checked, %d received, %d still short, %d with a booking in progress",
				id, result.Checked, len(result.Fixed), len(result.StillShort), len(result.BookingInProgress))
			for _, po := range result.Fixed {
				if *dryRun {
					log.Printf("  would mark received: %s", po)
				} else {
					log.Printf("  marked received: %s", po)
				}
			}
			for _, po := range result.BookingInProgress {
				log.Printf("  skipped, delivery being booked: %s", po)
			}
		}
		if err != nil {
			log.Printf("tenant %s: %v", id, err)
			failed = true
		}
	}
	if failed {
		log.Fatal("finished with errors -- fix them and run again (already repaired POs are skipped)")
	}
}
