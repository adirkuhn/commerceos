package main

import (
	"context"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	messagecenterapp "github.com/adirkuhn/commerceos/application/messagecenter"
	notificationsapp "github.com/adirkuhn/commerceos/application/notifications"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	procurementapp "github.com/adirkuhn/commerceos/application/procurement"
	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	schedulerapp "github.com/adirkuhn/commerceos/application/scheduler"
	shippingapp "github.com/adirkuhn/commerceos/application/shipping"
	whatsappapp "github.com/adirkuhn/commerceos/application/whatsapp"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
	bolinfra "github.com/adirkuhn/commerceos/infrastructure/bol"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/infrastructure/evolutionapi"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/noop"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/smtp"
	myparcelinfra "github.com/adirkuhn/commerceos/infrastructure/myparcel"
	"github.com/adirkuhn/commerceos/infrastructure/pdf"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
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

func main() {
	cfg := config.Load()

	db, err := db.Open(cfg.DBDsn, 10)
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
	expireBatchUC := invapp.NewExpireBatchUseCase(db, batchRepo, outboxRepo)
	promotionSKURepo := repositories.NewPromotionSKURepo(db)
	setPromotionSalePriceUC := pmsapp.NewSetPromotionSalePriceUseCase(db, variantRepo, skuRepo, outboxRepo, priceHistoryRepo)
	applyPromotionSKUUC := apppromotions.NewApplyPromotionSKUUseCase(db, skuRepo, setPromotionSalePriceUC, promotionSKURepo)
	revertPromotionSKUUC := apppromotions.NewRevertPromotionSKUUseCase(db, setPromotionSalePriceUC, promotionSKURepo)
	finishClearanceLinesUC := apppromotions.NewFinishClearanceLinesUseCase(
		db, invapp.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
		promotionSKURepo, repositories.NewPromotionRepo(db), revertPromotionSKUUC,
	)
	jobRepo := repositories.NewScheduledJobRepo(db)

	costLedgerRecorder := costLedgerRecorderAdapter{uc: pmsapp.NewRecordCostLedgerMovementUseCase(
		db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), invapp.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
	)}
	stockRecorderUC := invapp.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo, costLedgerRecorder)
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
		schedulerDomain.JobPromotionActivation:    schedulerapp.NewRunPromotionActivationUseCase(db, promotionSKURepo, applyPromotionSKUUC, revertPromotionSKUUC, promotionSKURepo, finishClearanceLinesUC),
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
	// Notifications honour each tenant's delivery preferences (bell and/or
	// email); the email half is sent by NotificationEmailConsumer below.
	notifyUC := notificationsapp.NewDeliverNotificationUseCase(db, notificationRepo, repositories.NewNotificationPreferencesRepo(db), outboxRepo)

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
	// syncStockUC is built here (moved up from its stockChangedConsumer
	// wiring further below) so syncProductUC can use it as its stock
	// catch-up dependency for reactivation after a gated period -- see
	// SyncProductToChannelsUseCase's doc comment.
	syncStockUC := channelsapp.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, syncLogRepo, connRepo, wooFactory, stockLevelUC, cfg.IntegrationsEncryptionSecret, pushBolUC,
	)
	syncProductUC := channelsapp.NewSyncProductToChannelsUseCase(
		db, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret, pushUC, mediaRepo, resolveCategoryLinkUC, pushBolUC, pushBolUC, syncStockUC,
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
	if sub := subscribeConsumer(js, events.ProductChanged{}.EventName(), "product-changed-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, productChangedConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	deleteProductFromChannelsUC := channelsapp.NewDeleteProductFromChannelsUseCase(db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret)
	productDeletedConsumer := consumers.NewProductDeletedConsumer(processedEventRepo, deleteProductFromChannelsUC, js)
	if sub := subscribeConsumer(js, events.ProductDeleted{}.EventName(), "product-deleted-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, productDeletedConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	variantUnlistedConsumer := consumers.NewVariantUnlistedConsumer(processedEventRepo, channelsapp.NewDeleteVariantFromChannelsUseCase(
		channelsapp.NewWooConnectionClients(db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret),
	), js)
	if sub := subscribeConsumer(js, events.VariantUnlisted{}.EventName(), "variant-unlisted-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, variantUnlistedConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	stockChangedConsumer := consumers.NewInventoryStockChangedConsumer(processedEventRepo, syncStockUC, js)
	if sub := subscribeConsumer(js, events.InventoryStockChanged{}.EventName(), "inventory-stock-changed-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, stockChangedConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	clearancePromotionStockConsumer := consumers.NewClearancePromotionStockConsumer(processedEventRepo, finishClearanceLinesUC, js)
	if sub := subscribeConsumer(js, events.InventoryStockChanged{}.EventName(), "inventory-stock-changed-clearance-promotions",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, clearancePromotionStockConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	pushSalesOrderStatusUC := channelsapp.NewPushSalesOrderStatusToChannelsUseCase(db, repositories.NewSalesOrderRepo(db), repositories.NewShipmentRepo(db), repositories.NewShippingCarrierOverrideRepo(db), connRepo, wooFactory, cfg.IntegrationsEncryptionSecret)
	salesOrderStatusChangedConsumer := consumers.NewSalesOrderStatusChangedConsumer(processedEventRepo, pushSalesOrderStatusUC, js)
	if sub := subscribeConsumer(js, events.SalesOrderStatusChanged{}.EventName(), "sales-order-status-changed-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, salesOrderStatusChangedConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

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
		db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, allocateStockUC, stockRecorderUC, outboxRepo,
	)
	jobRunners[schedulerDomain.JobReconcileFulfilledOrders] = schedulerapp.NewReconcileFulfilledOrdersUseCase(db, salesOrderRepo, updateSalesOrderUC)

	notifyOrderStatusUC := whatsappapp.NewNotifyOrderStatusUseCase(db, salesOrderRepo, customerRepo, connRepo, evolutionapi.NewClientFactory(), cfg.IntegrationsEncryptionSecret)
	whatsAppOrderStatusConsumer := consumers.NewWhatsAppOrderStatusConsumer(processedEventRepo, notifyOrderStatusUC, js)
	if sub := subscribeConsumer(js, events.SalesOrderStatusChanged{}.EventName(), "whatsapp-order-status-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, whatsAppOrderStatusConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	ordersImportUC := channelsapp.NewImportWooOrdersUseCase(
		db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret,
		salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, customerRepo,
		allocateStockUC, updateSalesOrderUC,
	)
	customersImportUC := channelsapp.NewImportWooCustomersUseCase(db, connRepo, wooFactory, cfg.IntegrationsEncryptionSecret, customerRepo)
	runManualWooImportUC := channelsapp.NewRunManualWooImportUseCase(importUC, customersImportUC, ordersImportUC)

	notifyShipmentDeliveryUC := whatsappapp.NewNotifyShipmentDeliveryUseCase(
		db, salesOrderRepo, customerRepo, connRepo, evolutionapi.NewClientFactory(), cfg.IntegrationsEncryptionSecret,
	)
	shipmentReconcileUC := shippingapp.NewReconcileShipmentsUseCase(
		db, repositories.NewShipmentRepo(db), repositories.NewShipmentStatusEventRepo(db),
		connRepo, salesOrderRepo, myparcelinfra.NewCarrierClientFactory(), cfg.IntegrationsEncryptionSecret,
		outboxRepo,
	)

	applyShipmentFulfillmentSignalUC := shippingapp.NewApplyShipmentFulfillmentSignalUseCase(db, salesOrderRepo, updateSalesOrderUC, notifyShipmentDeliveryUC)
	shipmentFulfillmentSignalConsumer := consumers.NewShipmentFulfillmentSignalConsumer(processedEventRepo, applyShipmentFulfillmentSignalUC, js)
	if sub := subscribeConsumer(js, events.ShipmentFulfillmentSignal{}.EventName(), "shipment-fulfillment-signal-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, shipmentFulfillmentSignalConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

	// Message Center automation engine -- one AutomationEngineConsumer
	// instance per known trigger subject (see
	// domain/messagecenter.KnownTriggerSubjects), each its own JetStream
	// subscription/durable/processed-events key, same
	// "independent consumer per subject" convention as every consumer
	// above, but sharing one handler implementation since the subject
	// itself is the variable part here (see AutomationEngineConsumer's own
	// doc comment). Email uses the same optional SMTP client as
	// cmd/api/main.go's storefront login emails -- unset SMTP_HOST means
	// every automation email silently no-ops (noop.Client) rather than the
	// scheduler failing to start.
	var automationEmailClient messaging.Client = noop.Client{}
	if cfg.SMTPHost != "" {
		automationEmailClient = smtp.NewClient(smtp.Config{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		})
	}
	evaluateAutomationRulesUC := messagecenterapp.NewEvaluateAutomationRulesUseCase(
		db, repositories.NewAutomationRuleRepo(db), repositories.NewMessageTemplateRepo(db),
		salesOrderRepo, customerRepo, connRepo, evolutionapi.NewClientFactory(), cfg.IntegrationsEncryptionSecret,
		smtp.NewClientFactory(), automationEmailClient,
	)
	for _, subject := range mcDomain.KnownTriggerSubjects {
		automationEngineConsumer := consumers.NewAutomationEngineConsumer(subject, processedEventRepo, evaluateAutomationRulesUC, js)
		// JetStream durable names may not contain "." (see nats.go's
		// checkConsumerName) -- subject itself keeps its dots (that's the
		// real NATS subject every event is published under), only the
		// durable name is sanitized.
		durable := "automation-engine-" + strings.ReplaceAll(subject, ".", "-")
		if sub := subscribeConsumer(js, subject, durable,
			consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, automationEngineConsumer.Handle); sub != nil {
			defer sub.Unsubscribe()
		}
	}

	// Email half of notifications (see DeliverNotificationUseCase): same
	// per-tenant email connection / platform SMTP fallback as automations.
	sendNotificationEmailUC := notificationsapp.NewSendNotificationEmailUseCase(
		db, connRepo, cfg.IntegrationsEncryptionSecret, smtp.NewClientFactory(), automationEmailClient, cfg.FrontendBaseURL,
	)
	notificationEmailConsumer := consumers.NewNotificationEmailConsumer(processedEventRepo, sendNotificationEmailUC, js)
	if sub := subscribeConsumer(js, events.NotificationEmailRequested{}.EventName(), "notification-email-scheduler",
		consumers.MaxDeliverBeforeDeadLetter, consumers.BackoffSchedule, notificationEmailConsumer.Handle); sub != nil {
		defer sub.Unsubscribe()
	}

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
