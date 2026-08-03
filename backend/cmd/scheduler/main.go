package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// pollInterval is how often the scheduler checks for due schedules — not
// the schedules' own cadence (that's SyncSchedule.IntervalMinutes). Keeping
// this well under the shortest configurable interval (15m) means a due job
// never waits much past its NextRunAt.
const pollInterval = 60 * time.Second

func main() {
	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	connRepo := repositories.NewConnectionRepo(db)
	wooFactory := woocommerce.NewFactory()
	categoryRepo := repositories.NewCategoryRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)

	stockRecorderUC := invapp.NewRecordStockMovementUseCase(db, movementRepo)
	stockLevelUC := invapp.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo)
	pushUC := channelsapp.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		productRepo, variantRepo, skuRepo, listingRepo, stockLevelUC,
	)

	rs := buildRunners(
		channelsapp.NewImportWooProductsUseCase(
			db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
			categoryRepo, productRepo, variantRepo, skuRepo, listingRepo, stockRecorderUC,
		),
		channelsapp.NewRunScheduledProductExportUseCase(db, productRepo, listingRepo, pushUC),
		channelsapp.NewImportWooStockLevelsUseCase(
			db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
			variantRepo, skuRepo, listingRepo, stockLevelUC, stockRecorderUC,
		),
		channelsapp.NewExportWooStockLevelsUseCase(
			db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
			variantRepo, skuRepo, listingRepo, stockLevelUC,
		),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Println("scheduler started")
	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler shutting down")
			return
		case <-ticker.C:
			runDue(ctx, db, scheduleRepo, rs)
		}
	}
}

// runDue polls for due schedules across every tenant — a system-level,
// cross-tenant query, so it runs under tenant.WithSuperAdmin (sets
// app.current_tenant = 'bypass_rls'), the same mechanism the JWT
// super_admin role uses. Never crashes the loop: a failure here or in any
// one schedule's run is logged and the next tick tries again.
func runDue(ctx context.Context, db *gorm.DB, scheduleRepo channelsDomain.SyncScheduleRepository, rs runners) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var due []channelsDomain.SyncSchedule
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		due, err = scheduleRepo.ListDue(ctx, time.Now())
		return err
	})
	if err != nil {
		log.Printf("list due schedules: %v", err)
		return
	}

	for _, s := range due {
		runOne(adminCtx, db, scheduleRepo, rs, s)
	}
}

// runOne dispatches a single due schedule under its own tenant's identity
// (SuperAdmin keeps RLS bypassed for the query itself, but IDFromContext
// must still resolve to the schedule's real TenantID so any rows the
// underlying use case creates — imported products, stock movements — are
// stamped with the correct tenant, not blank).
func runOne(adminCtx context.Context, db *gorm.DB, scheduleRepo channelsDomain.SyncScheduleRepository, rs runners, s channelsDomain.SyncSchedule) {
	runCtx := tenant.WithTenantID(adminCtx, s.TenantID)

	ranAt := time.Now()
	status := "success"
	errMsg := ""
	if err := rs.dispatch(runCtx, s); err != nil {
		status = "error"
		errMsg = err.Error()
		log.Printf("sync schedule %s (%s/%s): %v", s.ID, s.EntityType, s.Direction, err)
	}

	nextRunAt := ranAt.Add(time.Duration(s.IntervalMinutes) * time.Minute)
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return scheduleRepo.RecordRunResult(ctx, s.ID, ranAt, status, errMsg, nextRunAt)
	})
	if err != nil {
		log.Printf("record run result for schedule %s: %v", s.ID, err)
	}
}
