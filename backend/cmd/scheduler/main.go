package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	notificationsapp "github.com/adirkuhn/commerceos/application/notifications"
	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/storage"
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

// buildFileStorage mirrors cmd/api/main.go's helper of the same name —
// each binary wires its own dependencies (no shared main-package library
// between commands), matching this codebase's existing convention.
func buildFileStorage(cfg config.Config) pmsDomain.FileStorage {
	switch cfg.StorageDriver {
	case "s3":
		if cfg.S3Endpoint == "" || cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" || cfg.S3Bucket == "" || cfg.S3PublicBaseURL == "" {
			log.Fatalf("storage: STORAGE_DRIVER=s3 requires S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY, S3_BUCKET, S3_PUBLIC_BASE_URL")
		}
		client, err := minio.New(cfg.S3Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
			Secure: cfg.S3UseSSL,
		})
		if err != nil {
			log.Fatalf("storage: minio client: %v", err)
		}
		return storage.NewS3Storage(client, cfg.S3Bucket, cfg.S3PublicBaseURL)
	case "local":
		return storage.NewLocalStorage(cfg.MediaUploadDir, cfg.PublicAPIBaseURL+"/api/v1/uploads")
	default:
		log.Fatalf("storage: unknown STORAGE_DRIVER %q (want \"local\" or \"s3\")", cfg.StorageDriver)
		return nil
	}
}

func main() {
	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Drain()
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}
	if err := publishers.EnsureStream(js); err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	connRepo := repositories.NewConnectionRepo(db)
	wooFactory := woocommerce.NewFactory()
	categoryRepo := repositories.NewCategoryRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)
	syncLogRepo := repositories.NewSyncLogRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	importJobRepo := repositories.NewImportJobRepo(db)
	notificationRepo := repositories.NewNotificationRepo(db)
	mediaRepo := repositories.NewMediaRepo(db)
	fileStorage := buildFileStorage(cfg)
	uploadMediaUC := pmsapp.NewUploadMediaUseCase(db, productRepo, variantRepo, mediaRepo, fileStorage)

	stockRecorderUC := invapp.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo)
	stockLevelUC := invapp.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo)
	pushUC := channelsapp.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, stockLevelUC,
	)
	importUC := channelsapp.NewImportWooProductsUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		categoryRepo, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, stockRecorderUC, uploadMediaUC,
	)
	notifyUC := notificationsapp.NewCreateNotificationUseCase(db, notificationRepo)

	processedEventRepo := repositories.NewProcessedEventRepo(db)
	syncProductUC := channelsapp.NewSyncProductToChannelsUseCase(
		db, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
	)
	productChangedConsumer := consumers.NewProductChangedConsumer(processedEventRepo, syncProductUC)
	// Durable name persists this subscription's delivery cursor in
	// JetStream across scheduler restarts — a redeploy resumes from where
	// it left off instead of replaying (or losing) everything.
	productChangedSub, err := js.Subscribe(events.ProductChanged{}.EventName(), productChangedConsumer.Handle,
		nats.Durable("product-changed-scheduler"), nats.ManualAck())
	if err != nil {
		log.Fatalf("subscribe pms.product.changed: %v", err)
	}
	defer productChangedSub.Unsubscribe()

	syncStockUC := channelsapp.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, syncLogRepo, connRepo, wooFactory, stockLevelUC, cfg.IntegrationsEncryptionSecret,
	)
	stockChangedConsumer := consumers.NewInventoryStockChangedConsumer(processedEventRepo, syncStockUC)
	stockChangedSub, err := js.Subscribe(events.InventoryStockChanged{}.EventName(), stockChangedConsumer.Handle,
		nats.Durable("inventory-stock-changed-scheduler"), nats.ManualAck())
	if err != nil {
		log.Fatalf("subscribe inventory.stock.changed: %v", err)
	}
	defer stockChangedSub.Unsubscribe()

	rs := buildRunners(
		importUC,
		channelsapp.NewRunScheduledProductExportUseCase(db, productRepo, listingRepo, pushUC),
		channelsapp.NewImportWooStockLevelsUseCase(
			db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
			variantRepo, skuRepo, listingRepo, syncLogRepo, stockLevelUC, stockRecorderUC,
		),
		channelsapp.NewExportWooStockLevelsUseCase(
			db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
			variantRepo, skuRepo, listingRepo, syncLogRepo, stockLevelUC,
		),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// A job can only be "running" while some scheduler process is actively
	// inside runImportJob. If the previous process died mid-run (crash,
	// redeploy) before reaching a terminal state, the row is stuck forever
	// — and since Create's double-job guard checks pending/running, that
	// permanently blocks any further import for the connection. Recover
	// once at startup, before polling begins.
	recoverOrphanedImportJobs(ctx, db, importJobRepo)

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

// recoverOrphanedImportJobs fails every job left "running" by a scheduler
// process that never got to mark it terminal — see the call site's comment
// for why this matters. Cross-tenant, so it runs under SuperAdmin like the
// rest of this file's polling functions. A failure here is logged, not
// fatal — worst case, a stuck job just waits for the next restart.
func recoverOrphanedImportJobs(ctx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var recovered int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		recovered, err = jobRepo.MarkOrphanedRunningAsFailed(ctx, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned import jobs: %v", err)
		return
	}
	if recovered > 0 {
		log.Printf("recovered %d orphaned import job(s) left running by a previous scheduler process", recovered)
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
