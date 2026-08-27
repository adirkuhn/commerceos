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
	appoms "github.com/adirkuhn/commerceos/application/oms"
	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	procurementapp "github.com/adirkuhn/commerceos/application/procurement"
	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	schedulerapp "github.com/adirkuhn/commerceos/application/scheduler"
	shippingapp "github.com/adirkuhn/commerceos/application/shipping"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
	bolinfra "github.com/adirkuhn/commerceos/infrastructure/bol"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	myparcelinfra "github.com/adirkuhn/commerceos/infrastructure/myparcel"
	"github.com/adirkuhn/commerceos/infrastructure/pdf"
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

// documentParsingJobPollInterval mirrors importJobPollInterval: a supplier
// document upload is also user-triggered and should start parsing promptly.
const documentParsingJobPollInterval = 5 * time.Second

// receivingParsingJobPollInterval mirrors documentParsingJobPollInterval: a
// supplier invoice upload is also user-triggered and should start parsing
// promptly.
const receivingParsingJobPollInterval = 5 * time.Second

// maxJobRetries caps how many times the startup recovery pass will requeue
// a job found stuck "running" (crash, redeploy) back to pending instead of
// failing it outright — see recoverOrphanedImportJobs and its two
// procurement siblings. Bounds a job whose own execution reliably crashes
// the scheduler process from requeuing itself forever.
const maxJobRetries = 3

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
	if err := publishers.EnsureDLQStream(js); err != nil {
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
	warehouseRepo := repositories.NewWarehouseRepo(db)
	costPriceHistoryRepo := repositories.NewSKUCostPriceHistoryRepo(db)
	priceHistoryRepo := repositories.NewSKUPriceHistoryRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	importJobRepo := repositories.NewImportJobRepo(db)
	notificationRepo := repositories.NewNotificationRepo(db)
	mediaRepo := repositories.NewMediaRepo(db)
	fileStorage := buildFileStorage(cfg)
	uploadMediaUC := pmsapp.NewUploadMediaUseCase(db, productRepo, variantRepo, mediaRepo, fileStorage)
	expireSalePriceUC := pmsapp.NewExpireSKUSalePriceUseCase(db, variantRepo, skuRepo, outboxRepo, priceHistoryRepo)
	expireBatchUC := invapp.NewExpireBatchUseCase(db, batchRepo)
	promotionSKURepo := repositories.NewPromotionSKURepo(db)
	setPromotionSalePriceUC := pmsapp.NewSetPromotionSalePriceUseCase(db, variantRepo, skuRepo, outboxRepo, priceHistoryRepo)
	applyPromotionSKUUC := apppromotions.NewApplyPromotionSKUUseCase(db, skuRepo, setPromotionSalePriceUC, promotionSKURepo)
	revertPromotionSKUUC := apppromotions.NewRevertPromotionSKUUseCase(db, setPromotionSalePriceUC, promotionSKURepo)
	jobRepo := repositories.NewScheduledJobRepo(db)

	stockRecorderUC := invapp.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	stockLevelUC := invapp.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo, settingsRepo)
	// lineItemRepo is constructed here (earlier than the rest of procurement's
	// wiring below) because recalculateCostPricesUC needs it too.
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	recalculateCostPricesUC := pmsapp.NewRecalculateCostPricesUseCase(
		db, skuRepo, variantRepo, productRepo, categoryRepo, settingsRepo, costPriceHistoryRepo,
		invapp.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
		procurementapp.NewListReceivedCostLinesForSKUUseCase(db, lineItemRepo),
	)
	jobRunners := map[string]schedulerapp.JobRunner{
		schedulerDomain.JobExpiredSalePrices:      schedulerapp.NewRunExpiredSalePricesUseCase(db, skuRepo, expireSalePriceUC),
		schedulerDomain.JobExpiredBatches:         schedulerapp.NewRunExpiredBatchesUseCase(db, batchRepo, expireBatchUC),
		schedulerDomain.JobCostPriceRecalculation: schedulerapp.NewRunCostPriceRecalculationUseCase(db, skuRepo, recalculateCostPricesUC),
		schedulerDomain.JobPromotionActivation:    schedulerapp.NewRunPromotionActivationUseCase(db, promotionSKURepo, applyPromotionSKUUC, revertPromotionSKUUC),
	}
	runJobNowUC := schedulerapp.NewRunJobNowUseCase(db, jobRepo, jobRunners)
	categoryLinkRepo := repositories.NewExternalCategoryLinkRepo(db)
	resolveCategoryLinkUC := channelsapp.NewResolveCategoryLinkUseCase(db, categoryLinkRepo, categoryRepo)
	pushUC := channelsapp.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, stockLevelUC, mediaRepo, resolveCategoryLinkUC,
	)
	taxClassRepo := repositories.NewTaxClassRepo(db)
	taxRateRepo := repositories.NewTaxRateRepo(db)
	taxLinkRepo := repositories.NewExternalTaxClassLinkRepo(db)
	importUC := channelsapp.NewImportWooProductsUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		categoryRepo, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, stockRecorderUC, uploadMediaUC,
		taxClassRepo, taxRateRepo, taxLinkRepo,
	)
	notifyUC := notificationsapp.NewCreateNotificationUseCase(db, notificationRepo)

	poRepo := repositories.NewPurchaseOrderRepo(db)
	documentRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	parsingJobRepo := repositories.NewDocumentParsingJobRepo(db)
	aliasRepo := repositories.NewSupplierProductAliasRepo(db)
	matchUC := procurementapp.NewMatchExtractedLineItemsUseCase(skuRepo, productRepo, variantRepo, aliasRepo)
	pdfExtractor := pdf.NewGxPDFExtractor()
	// aiParserResolver looks up each job's own tenant's AIConnection (see
	// integrations.AIConnection) and builds the matching adapter -- resolves
	// to nil when the tenant hasn't configured a provider, so PDFs then fail
	// clearly with procurement.ErrAIParsingNotConfigured instead of the job
	// crashing; CSV/XLSX parsing and exact-SKU/alias matching both work with
	// no AI configured at all.
	aiConnRepo := repositories.NewAIConnectionRepo(db)
	aiParserResolver := ai.NewParserResolver(db, aiConnRepo, cfg.IntegrationsEncryptionSecret)
	runDocumentParsingJobUC := procurementapp.NewRunDocumentParsingJobUseCase(
		db, poRepo, lineItemRepo, documentRepo, matchUC, fileStorage, pdfExtractor, aiParserResolver,
	)

	receivingRecordRepo := repositories.NewReceivingRecordRepo(db)
	receivingJobRepo := repositories.NewReceivingParsingJobRepo(db)
	discrepancyRepo := repositories.NewReceivingDiscrepancyRepo(db)
	runReceivingParsingJobUC := procurementapp.NewRunReceivingParsingJobUseCase(
		db, receivingRecordRepo, lineItemRepo, discrepancyRepo, fileStorage, pdfExtractor, aiParserResolver,
	)

	processedEventRepo := repositories.NewProcessedEventRepo(db)
	bolOverrideRepo := repositories.NewBolProductOverrideRepo(db)
	bolContentRepo := repositories.NewBolProductContentRepo(db)
	pushBolUC := channelsapp.NewPushProductToBolUseCase(
		db, connRepo, bolinfra.NewClientFactory(), cfg.IntegrationsEncryptionSecret, productRepo, variantRepo, skuRepo, listingRepo, bolOverrideRepo, bolContentRepo, syncLogRepo, stockLevelUC, notifyUC,
	)
	syncProductUC := channelsapp.NewSyncProductToChannelsUseCase(
		db, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret, pushUC, mediaRepo, resolveCategoryLinkUC, pushBolUC,
	)
	salesChannelRepo := repositories.NewSalesChannelRepo(db)
	salesChannelOverrideRepo := repositories.NewSalesChannelProductOverrideRepo(db)
	enrollProductUC := channelsapp.NewEnrollProductInChannelsUseCase(db, productRepo, salesChannelRepo, salesChannelOverrideRepo, pushUC, pushBolUC)
	productChangedConsumer := consumers.NewProductChangedConsumer(processedEventRepo, enrollProductUC, syncProductUC, js)
	// Durable name persists this subscription's delivery cursor in
	// JetStream across scheduler restarts — a redeploy resumes from where
	// it left off instead of replaying (or losing) everything. MaxDeliver
	// caps redelivery at consumers.MaxDeliverBeforeDeadLetter -- past that,
	// nakOrDeadLetter routes the message to the DLQ stream instead of
	// nacking it into another attempt already proven to fail. BackOff
	// paces those redeliveries instead of hammering the handler at flat
	// ~30s (AckWait's default) intervals.
	productChangedSub, err := js.Subscribe(events.ProductChanged{}.EventName(), productChangedConsumer.Handle,
		nats.Durable("product-changed-scheduler"), nats.ManualAck(),
		nats.MaxDeliver(consumers.MaxDeliverBeforeDeadLetter), nats.BackOff(consumers.BackoffSchedule))
	if err != nil {
		log.Fatalf("subscribe pms.product.changed: %v", err)
	}
	defer productChangedSub.Unsubscribe()

	deleteProductFromChannelsUC := channelsapp.NewDeleteProductFromChannelsUseCase(db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret)
	productDeletedConsumer := consumers.NewProductDeletedConsumer(processedEventRepo, deleteProductFromChannelsUC, js)
	productDeletedSub, err := js.Subscribe(events.ProductDeleted{}.EventName(), productDeletedConsumer.Handle,
		nats.Durable("product-deleted-scheduler"), nats.ManualAck(),
		nats.MaxDeliver(consumers.MaxDeliverBeforeDeadLetter), nats.BackOff(consumers.BackoffSchedule))
	if err != nil {
		log.Fatalf("subscribe pms.product.deleted: %v", err)
	}
	defer productDeletedSub.Unsubscribe()

	syncStockUC := channelsapp.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, syncLogRepo, connRepo, wooFactory, stockLevelUC, cfg.IntegrationsEncryptionSecret, pushBolUC,
	)
	stockChangedConsumer := consumers.NewInventoryStockChangedConsumer(processedEventRepo, syncStockUC, js)
	stockChangedSub, err := js.Subscribe(events.InventoryStockChanged{}.EventName(), stockChangedConsumer.Handle,
		nats.Durable("inventory-stock-changed-scheduler"), nats.ManualAck(),
		nats.MaxDeliver(consumers.MaxDeliverBeforeDeadLetter), nats.BackOff(consumers.BackoffSchedule))
	if err != nil {
		log.Fatalf("subscribe inventory.stock.changed: %v", err)
	}
	defer stockChangedSub.Unsubscribe()

	// jobRunners was built before syncProductUC/syncStockUC existed --
	// added here instead of at its original declaration. Safe to mutate
	// after being handed to runJobNowUC: that's the same map by reference,
	// and nothing reads from it until the poll loop starts further down.
	resyncProductToChannelsUC := channelsapp.NewResyncProductToChannelsUseCase(db, variantRepo, skuRepo, syncProductUC, syncStockUC, enrollProductUC)
	jobRunners[schedulerDomain.JobResyncAllProductsToChannels] = schedulerapp.NewRunResyncProductsToChannelsUseCase(db, listingRepo, resyncProductToChannelsUC)

	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	salesOrderLineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	salesOrderLinkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	customerRepo := repositories.NewCustomerRepo(db)
	allocateStockUC := invapp.NewAllocateSaleStockUseCase(db, movementRepo, batchRepo, outboxRepo)
	updateSalesOrderUC := appoms.NewUpdateSalesOrderUseCase(
		db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, allocateStockUC, stockRecorderUC,
	)

	ordersImportUC := channelsapp.NewImportWooOrdersUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, customerRepo,
		allocateStockUC, updateSalesOrderUC,
	)
	customersImportUC := channelsapp.NewImportWooCustomersUseCase(db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret, customerRepo)
	runManualWooImportUC := channelsapp.NewRunManualWooImportUseCase(importUC, customersImportUC, ordersImportUC)

	shipmentReconcileUC := shippingapp.NewReconcileShipmentsUseCase(
		db, repositories.NewShipmentRepo(db), repositories.NewShipmentStatusEventRepo(db),
		connRepo, salesOrderRepo, myparcelinfra.NewClientFactory(), cfg.IntegrationsEncryptionSecret,
	)

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
		ordersImportUC,
		shipmentReconcileUC,
		channelsapp.NewImportBolOrdersUseCase(
			db, connRepo, bolinfra.NewClientFactory(), cfg.IntegrationsEncryptionSecret,
			salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo,
			allocateStockUC, updateSalesOrderUC,
		),
		channelsapp.NewRunScheduledBolExportUseCase(db, productRepo, salesChannelRepo, salesChannelOverrideRepo, pushBolUC),
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
	recoverOrphanedDocumentParsingJobs(ctx, db, parsingJobRepo)
	recoverOrphanedReceivingParsingJobs(ctx, db, receivingJobRepo)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	importJobTicker := time.NewTicker(importJobPollInterval)
	defer importJobTicker.Stop()
	documentParsingJobTicker := time.NewTicker(documentParsingJobPollInterval)
	defer documentParsingJobTicker.Stop()
	receivingParsingJobTicker := time.NewTicker(receivingParsingJobPollInterval)
	defer receivingParsingJobTicker.Stop()

	log.Println("scheduler started")
	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler shutting down")
			return
		case <-ticker.C:
			runDue(ctx, db, scheduleRepo, rs)
			runDueJobs(ctx, db, jobRepo, runJobNowUC)
		case <-importJobTicker.C:
			runPendingImportJobs(ctx, db, importJobRepo, runManualWooImportUC, notifyUC)
		case <-documentParsingJobTicker.C:
			runPendingDocumentParsingJobs(ctx, db, parsingJobRepo, runDocumentParsingJobUC, notifyUC)
		case <-receivingParsingJobTicker.C:
			runPendingReceivingParsingJobs(ctx, db, receivingJobRepo, runReceivingParsingJobUC, notifyUC)
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
	cursorAt, err := rs.dispatch(runCtx, s)
	if err != nil {
		status = "error"
		errMsg = err.Error()
		log.Printf("sync schedule %s (%s/%s): %v", s.ID, s.EntityType, s.Direction, err)
	}
	// cursorAt nil means the runner has no cursor of its own separate from
	// wall-clock time (every runner except orders import) -- see runner's
	// doc comment in dispatch.go.
	if cursorAt == nil {
		cursorAt = &ranAt
	}

	nextRunAt := ranAt.Add(time.Duration(s.IntervalMinutes) * time.Minute)
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return scheduleRepo.RecordRunResult(ctx, s.ID, ranAt, status, errMsg, nextRunAt, *cursorAt, status == "success")
	})
	if err != nil {
		log.Printf("record run result for schedule %s: %v", s.ID, err)
	}
}

// runDueJobs polls domain/scheduler's global cron-job registry (see
// migration 085_scheduled_jobs.sql) the same way runDue above polls
// SyncSchedule -- a system-level, cross-tenant query under SuperAdmin.
// Each due job is dispatched through the exact same RunJobNowUseCase a
// super_admin's manual "run now" on the admin page uses, so "last run"
// stays accurate and consistent regardless of which path actually
// triggered it. Never crashes the loop: a failure on one job (or listing
// due jobs at all) is logged and the next tick tries again.
func runDueJobs(ctx context.Context, db *gorm.DB, jobRepo schedulerDomain.JobRepository, runJobNowUC *schedulerapp.RunJobNowUseCase) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var due []schedulerDomain.Job
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		due, err = jobRepo.ListDue(ctx, time.Now())
		return err
	})
	if err != nil {
		log.Printf("list due scheduled jobs: %v", err)
		return
	}

	for _, j := range due {
		if err := runJobNowUC.Execute(adminCtx, j.JobKey); err != nil {
			log.Printf("run scheduled job %s: %v", j.JobKey, err)
		}
	}
}

// recoverOrphanedImportJobs requeues (or, past maxJobRetries, fails) every
// job left "running" by a scheduler process that never got to mark it
// terminal — see the call site's comment for why this matters. Cross-tenant,
// so it runs under SuperAdmin like the rest of this file's polling
// functions. A failure here is logged, not fatal — worst case, a stuck job
// just waits for the next restart.
func recoverOrphanedImportJobs(ctx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxJobRetries, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned import jobs: %v", err)
		return
	}
	if requeued > 0 {
		log.Printf("requeued %d orphaned import job(s) left running by a previous scheduler process", requeued)
	}
	if failed > 0 {
		log.Printf("failed %d orphaned import job(s) that exhausted retries", failed)
	}
}

// runPendingImportJobs polls for pending ImportJobs across every tenant —
// same cross-tenant/SuperAdmin shape as runDue. Never crashes the loop: a
// failure here or in any one job's run is logged and the next tick tries
// again.
func runPendingImportJobs(
	ctx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository,
	runImportUC *channelsapp.RunManualWooImportUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
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
		runImportJob(adminCtx, db, jobRepo, runImportUC, notifyUC, job)
	}
}

// runImportJob runs a single pending job under its own tenant's identity
// (see runOne's comment for why) and always leaves it in a terminal state
// (completed/failed) plus a matching notification, even on failure.
// RunManualWooImportUseCase also bootstraps a full (stock-untouched) orders
// import the very first time this connection ever imports products — see
// its own doc comment; an orders-side failure there is folded into this
// job's own Errors list rather than failing the job, since the products
// import (the job's primary purpose) already succeeded.
func runImportJob(
	adminCtx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository,
	runImportUC *channelsapp.RunManualWooImportUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
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

	result, err := runImportUC.Execute(runCtx, channelsapp.RunManualWooImportInput{
		ConnectionID: job.ConnectionID, DefaultWarehouseID: job.DefaultWarehouseID, DefaultCurrency: job.DefaultCurrency,
		IsFirstImport: job.IsFirstImport,
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

	products := result.Products
	errs := products.Errors
	message := fmt.Sprintf("Imported %d products (%d already linked, %d skipped)", products.Imported, products.AlreadyLinked, products.SkippedType)
	if result.Customers != nil {
		errs = append(errs, result.Customers.Errors...)
		message += fmt.Sprintf(", %d customers", result.Customers.Created)
	}
	if result.Orders != nil {
		errs = append(errs, result.Orders.Errors...)
		message += fmt.Sprintf(", %d orders", result.Orders.Created)
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, products.Imported, products.AlreadyLinked, products.SkippedType, errs, finishedAt)
	}); markErr != nil {
		log.Printf("mark import job %s completed: %v", job.ID, markErr)
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeImportCompleted, "Import completed", message)
}

// recoverOrphanedDocumentParsingJobs mirrors recoverOrphanedImportJobs for
// DocumentParsingJob -- see that function's comment for the full rationale.
func recoverOrphanedDocumentParsingJobs(ctx context.Context, db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxJobRetries, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned document parsing jobs: %v", err)
		return
	}
	if requeued > 0 {
		log.Printf("requeued %d orphaned document parsing job(s) left running by a previous scheduler process", requeued)
	}
	if failed > 0 {
		log.Printf("failed %d orphaned document parsing job(s) that exhausted retries", failed)
	}
}

// runPendingDocumentParsingJobs mirrors runPendingImportJobs for
// DocumentParsingJob -- same cross-tenant/SuperAdmin poll shape.
func runPendingDocumentParsingJobs(
	ctx context.Context, db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository,
	runUC *procurementapp.RunDocumentParsingJobUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []procurementDomain.DocumentParsingJob
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	if err != nil {
		log.Printf("list pending document parsing jobs: %v", err)
		return
	}

	for _, job := range pending {
		runDocumentParsingJob(adminCtx, db, jobRepo, runUC, notifyUC, job)
	}
}

// runDocumentParsingJob runs a single pending job under its own tenant's
// identity and always leaves it in a terminal state (completed/failed) plus
// a matching notification, even on failure -- same shape as runImportJob.
func runDocumentParsingJob(
	adminCtx context.Context, db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository,
	runUC *procurementapp.RunDocumentParsingJobUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
	job procurementDomain.DocumentParsingJob,
) {
	runCtx := tenant.WithTenantID(adminCtx, job.TenantID)

	startedAt := time.Now()
	if err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, startedAt)
	}); err != nil {
		log.Printf("mark document parsing job %s running: %v", job.ID, err)
		return
	}

	needsReview, err := runUC.Execute(runCtx, job)
	finishedAt := time.Now()

	if err != nil {
		log.Printf("document parsing job %s: %v", job.ID, err)
		if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkFailed(ctx, job.ID, err.Error(), finishedAt)
		}); markErr != nil {
			log.Printf("mark document parsing job %s failed: %v", job.ID, markErr)
		}
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeDocumentParsingFailed, "Document parsing failed", err.Error())
		return
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, finishedAt)
	}); markErr != nil {
		log.Printf("mark document parsing job %s completed: %v", job.ID, markErr)
	}

	if needsReview {
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypePurchaseOrderNeedsReview, "Purchase order needs review",
			"Some line items couldn't be matched to a product automatically — review and link them.")
		return
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeDocumentParsingCompleted, "Document parsed",
		"All line items were matched to products automatically.")
}

// recoverOrphanedReceivingParsingJobs mirrors recoverOrphanedDocumentParsingJobs
// for ReceivingParsingJob -- see recoverOrphanedImportJobs's comment for the
// full rationale.
func recoverOrphanedReceivingParsingJobs(ctx context.Context, db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxJobRetries, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned receiving parsing jobs: %v", err)
		return
	}
	if requeued > 0 {
		log.Printf("requeued %d orphaned receiving parsing job(s) left running by a previous scheduler process", requeued)
	}
	if failed > 0 {
		log.Printf("failed %d orphaned receiving parsing job(s) that exhausted retries", failed)
	}
}

// runPendingReceivingParsingJobs mirrors runPendingDocumentParsingJobs for
// ReceivingParsingJob -- same cross-tenant/SuperAdmin poll shape.
func runPendingReceivingParsingJobs(
	ctx context.Context, db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository,
	runUC *procurementapp.RunReceivingParsingJobUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []procurementDomain.ReceivingParsingJob
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	if err != nil {
		log.Printf("list pending receiving parsing jobs: %v", err)
		return
	}

	for _, job := range pending {
		runReceivingParsingJob(adminCtx, db, jobRepo, runUC, notifyUC, job)
	}
}

// runReceivingParsingJob runs a single pending job under its own tenant's
// identity and always leaves it in a terminal state (completed/failed) plus
// a matching notification, even on failure -- same shape as runDocumentParsingJob.
func runReceivingParsingJob(
	adminCtx context.Context, db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository,
	runUC *procurementapp.RunReceivingParsingJobUseCase, notifyUC *notificationsapp.CreateNotificationUseCase,
	job procurementDomain.ReceivingParsingJob,
) {
	runCtx := tenant.WithTenantID(adminCtx, job.TenantID)

	startedAt := time.Now()
	if err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, startedAt)
	}); err != nil {
		log.Printf("mark receiving parsing job %s running: %v", job.ID, err)
		return
	}

	discrepancyCount, err := runUC.Execute(runCtx, job)
	finishedAt := time.Now()

	if err != nil {
		log.Printf("receiving parsing job %s: %v", job.ID, err)
		if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkFailed(ctx, job.ID, err.Error(), finishedAt)
		}); markErr != nil {
			log.Printf("mark receiving parsing job %s failed: %v", job.ID, markErr)
		}
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeReceivingParsingFailed, "Invoice parsing failed", err.Error())
		return
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, finishedAt)
	}); markErr != nil {
		log.Printf("mark receiving parsing job %s completed: %v", job.ID, markErr)
	}

	if discrepancyCount > 0 {
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeReceivingDiscrepanciesFound, "Receiving discrepancies found",
			fmt.Sprintf("The invoice didn't fully match the purchase order (%d discrepancies) — review and reconcile.", discrepancyCount))
		return
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeReceivingParsingCompleted, "Invoice parsed",
		"The invoice matched the purchase order exactly.")
}

// notifyBestEffort never aborts the caller — a notification failing to
// write shouldn't hide that the actual import job did complete or fail.
func notifyBestEffort(ctx context.Context, notifyUC *notificationsapp.CreateNotificationUseCase, t notificationsDomain.Type, title, message string) {
	if _, err := notifyUC.Execute(ctx, notificationsapp.CreateNotificationInput{Type: t, Title: title, Message: message}); err != nil {
		log.Printf("notify %s: %v", t, err)
	}
}
