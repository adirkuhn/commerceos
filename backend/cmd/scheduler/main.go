package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	notificationsapp "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
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

// importJobPollInterval is separate and much shorter: manual import jobs
// are user-triggered (the "Import" button) and should start promptly, not
// wait up to a full minute the way a periodic sync schedule can afford to.
const importJobPollInterval = 5 * time.Second

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
	importJobRepo := repositories.NewImportJobRepo(db)
	notificationRepo := repositories.NewNotificationRepo(db)

	stockRecorderUC := invapp.NewRecordStockMovementUseCase(db, movementRepo)
	stockLevelUC := invapp.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo)
	pushUC := channelsapp.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		productRepo, variantRepo, skuRepo, listingRepo, stockLevelUC,
	)
	importUC := channelsapp.NewImportWooProductsUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		categoryRepo, productRepo, variantRepo, skuRepo, listingRepo, stockRecorderUC,
	)
	notifyUC := notificationsapp.NewCreateNotificationUseCase(db, notificationRepo)

	rs := buildRunners(
		importUC,
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
	importJobTicker := time.NewTicker(importJobPollInterval)
	defer importJobTicker.Stop()

	log.Println("scheduler started")
	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler shutting down")
			return
		case <-ticker.C:
			runDue(ctx, db, scheduleRepo, rs)
		case <-importJobTicker.C:
			runPendingImportJobs(ctx, db, importJobRepo, importUC, notifyUC)
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

// runPendingImportJobs polls for pending ImportJobs across every tenant —
// same cross-tenant/SuperAdmin shape as runDue. Never crashes the loop: a
// failure here or in any one job's run is logged and the next tick tries
// again.
func runPendingImportJobs(
	ctx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository,
	importUC *channelsapp.ImportWooProductsUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []channelsDomain.ImportJob
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	if err != nil {
		log.Printf("list pending import jobs: %v", err)
		return
	}

	for _, job := range pending {
		runImportJob(adminCtx, db, jobRepo, importUC, notifyUC, job)
	}
}

// runImportJob runs a single pending job under its own tenant's identity
// (see runOne's comment for why) and always leaves it in a terminal state
// (completed/failed) plus a matching notification, even on failure.
func runImportJob(
	adminCtx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository,
	importUC *channelsapp.ImportWooProductsUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
	job channelsDomain.ImportJob,
) {
	runCtx := tenant.WithTenantID(adminCtx, job.TenantID)

	startedAt := time.Now()
	if err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, startedAt)
	}); err != nil {
		log.Printf("mark import job %s running: %v", job.ID, err)
		return
	}

	result, err := importUC.Execute(runCtx, channelsapp.ImportWooProductsInput{
		ConnectionID: job.ConnectionID, DefaultWarehouseID: job.DefaultWarehouseID, DefaultCurrency: job.DefaultCurrency,
	})
	finishedAt := time.Now()

	if err != nil {
		log.Printf("import job %s: %v", job.ID, err)
		if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkFailed(ctx, job.ID, err.Error(), finishedAt)
		}); markErr != nil {
			log.Printf("mark import job %s failed: %v", job.ID, markErr)
		}
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeImportFailed, "Import failed", err.Error())
		return
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, result.Imported, result.AlreadyLinked, result.SkippedType, result.Errors, finishedAt)
	}); markErr != nil {
		log.Printf("mark import job %s completed: %v", job.ID, markErr)
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeImportCompleted, "Import completed",
		fmt.Sprintf("Imported %d products (%d already linked, %d skipped)", result.Imported, result.AlreadyLinked, result.SkippedType))
}

// notifyBestEffort never aborts the caller — a notification failing to
// write shouldn't hide that the actual import job did complete or fail.
func notifyBestEffort(ctx context.Context, notifyUC *notificationsapp.CreateNotificationUseCase, t notificationsDomain.Type, title, message string) {
	if _, err := notifyUC.Execute(ctx, notificationsapp.CreateNotificationInput{Type: t, Title: title, Message: message}); err != nil {
		log.Printf("notify %s: %v", t, err)
	}
}
