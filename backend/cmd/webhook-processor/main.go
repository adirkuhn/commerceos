// cmd/webhook-processor is the cold-path deployable for inbound webhooks --
// separate from cmd/api (which only verifies+publishes) and from
// cmd/relay/cmd/scheduler (which handle the outbox-based domain-event
// path), mirroring this app's existing convention of one small process per
// concern. Multiple instances can run at once: they share a NATS queue
// group, so JetStream load-balances COMMERCEOS_WEBHOOKS across them.
package main

import (
	"context"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"log"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppayments "github.com/adirkuhn/commerceos/application/payments"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
	"github.com/adirkuhn/commerceos/infrastructure/carriers"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/infrastructure/evolutionapi"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/revolut"
	"github.com/adirkuhn/commerceos/infrastructure/stripe"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
)

// webhookQueueGroup is the NATS queue group name every webhook-processor
// instance subscribes under -- JetStream delivers each message to only one
// member of the group, which is what makes running several instances a
// horizontal scale-out rather than N-times redundant processing.
const webhookQueueGroup = "webhook-processor"

// webhookConsumerDurable names the durable JetStream consumer shared by
// the whole queue group, so a redeployed/restarted instance resumes from
// where the group left off instead of replaying (or losing) everything.
const webhookConsumerDurable = "webhook-processor"

// costLedgerRecorderAdapter bridges application/inventory's local
// costLedgerRecorder interface to application/pms's real
// RecordCostLedgerMovementUseCase -- lives here (not in either application
// package) because application/pms already imports application/inventory
// (see application/pms/recalculate_cost_prices.go's doc comment), so the
// reverse import would cycle.
type costLedgerRecorderAdapter struct {
	uc *apppms.RecordCostLedgerMovementUseCase
}

func (a costLedgerRecorderAdapter) Execute(ctx context.Context, in appinventory.CostLedgerMovementInput) error {
	return a.uc.Execute(ctx, apppms.RecordCostLedgerMovementInput{
		SKUID: in.SKUID, QuantityDelta: in.QuantityDelta,
		SourceType:    pmsDomain.SKUCostPriceLedgerSourceType(in.SourceType),
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

func main() {
	cfg := config.Load()

	db, err := db.Open(cfg.DBDsn, 10)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	nc, err := nats.Connect(cfg.NatsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Drain()
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}
	if err := publishers.EnsureWebhookStream(js); err != nil {
		log.Fatalf("jetstream: %v", err)
	}
	if err := publishers.EnsureWebhookDLQStream(js); err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	processedRepo := repositories.NewProcessedEventRepo(db)
	eventRepo := repositories.NewWebhookEventRepo(db)

	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	linkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	skuRepo := repositories.NewSKURepo(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)
	costLedgerRecorder := costLedgerRecorderAdapter{uc: apppms.NewRecordCostLedgerMovementUseCase(
		db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), appinventory.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
	)}
	recordMovement := appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo, costLedgerRecorder)
	allocateStock := appinventory.NewAllocateSaleStockUseCase(db, movementRepo, batchRepo, outboxRepo)
	updateSalesOrder := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, appintegrations.NewSalesOrderOrigins(repositories.NewSalesOrderOriginRepo(db)), lineItemRepo, linkRepo, skuRepo, allocateStock, recordMovement, outboxRepo)

	shipmentHandler := appshipping.NewHandleShipmentWebhookUseCase(
		db, repositories.NewShipmentRepo(db), repositories.NewShipmentStatusEventRepo(db),
		repositories.NewConnectionRepo(db), salesOrderRepo,
		carriers.NewDefaultRegistry(), cfg.IntegrationsEncryptionSecret, outboxRepo,
	)
	orderStatusHandler := appshipping.NewHandleOrderStatusWebhookUseCase(
		db, repositories.NewMyParcelFulfilmentOrderRepo(db), outboxRepo,
	)
	myParcelRouter := appshipping.NewMyParcelWebhookRouter(shipmentHandler, orderStatusHandler)
	forwardRuleRepo := repositories.NewWebhookForwardRuleRepo(db)
	forwardDeliveryRepo := repositories.NewWebhookForwardDeliveryRepo(db)

	lookupProductInfo := appwhatsapp.NewLookupProductInfoUseCase(
		db, repositories.NewSKURepo(db), repositories.NewVariantRepo(db), repositories.NewProductRepo(db),
		repositories.NewSalesChannelRepo(db), repositories.NewSalesChannelProductOverrideRepo(db),
		appinventory.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo, repositories.NewTenantSettingsRepo(db)),
		repositories.NewTenantSettingsRepo(db),
	)
	createSalesOrderUC := appoms.NewCreateSalesOrderUseCase(
		db, salesOrderRepo, appintegrations.NewSalesOrderOrigins(repositories.NewSalesOrderOriginRepo(db)), lineItemRepo, linkRepo, repositories.NewSKURepo(db), allocateStock, outboxRepo,
	).WithNumbers(repositories.NewSalesOrderNumberRepo(db))
	createCustomerUC := appcustomers.NewCreateCustomerUseCase(db, repositories.NewCustomerRepo(db), outboxRepo)
	generatePaymentLinkUC := appwhatsapp.NewGeneratePaymentLinkUseCase(
		db, repositories.NewSalesChannelRepo(db), repositories.NewConnectionRepo(db),
		stripe.NewClientFactory(), revolut.NewClientFactory(), cfg.IntegrationsEncryptionSecret, cfg.FrontendBaseURL,
	)
	orderFlowUC := appwhatsapp.NewOrderFlowUseCase(
		db, repositories.NewWhatsAppConversationRepo(db), repositories.NewSalesChannelRepo(db), lookupProductInfo, createSalesOrderUC, createCustomerUC,
		generatePaymentLinkUC, outboxRepo,
	)
	whatsappHandler := appwhatsapp.NewHandleInboundMessageUseCase(
		db, repositories.NewConnectionRepo(db), evolutionapi.NewClientFactory(), cfg.IntegrationsEncryptionSecret,
		lookupProductInfo, repositories.NewCustomerRepo(db), orderFlowUC, repositories.NewWhatsAppMessageRepo(db),
		ai.NewWhatsAppIntentParserResolver(db, repositories.NewAIConnectionRepo(db), cfg.IntegrationsEncryptionSecret),
		appwhatsapp.NewResolveSKUFromQueryUseCase(db, repositories.NewSKURepo(db), repositories.NewProductRepo(db), repositories.NewVariantRepo(db)), outboxRepo,
	)

	stripePaymentHandler := apppayments.NewWebhookDispatchHandler(db, salesOrderRepo, apppayments.NewStripeEventParser(), updateSalesOrder, "stripe")
	revolutPaymentHandler := apppayments.NewWebhookDispatchHandler(db, salesOrderRepo, apppayments.NewRevolutEventParser(), updateSalesOrder, "revolut")

	dispatchHandlers := map[string]appwebhooks.DispatchHandler{
		"myparcel": myParcelRouter,
		"whatsapp": whatsappHandler,
		"stripe":   stripePaymentHandler,
		"revolut":  revolutPaymentHandler,
	}
	processWebhookEventUC := appwebhooks.NewProcessWebhookEventUseCase(db, eventRepo, dispatchHandlers, forwardRuleRepo, forwardDeliveryRepo)
	webhookConsumer := consumers.NewWebhookConsumer(processedRepo, processWebhookEventUC, js)

	// "webhooks.received.*" -- one subscription covers every provider
	// (see infrastructure/webhookproviders for the registry cmd/api's hot
	// path verifies against); the consumer itself doesn't branch on
	// provider name.
	sub, err := js.QueueSubscribe("webhooks.received.*", webhookQueueGroup, consumers.OnlyRetriesFor(webhookConsumerDurable, webhookConsumer.Handle),
		nats.Durable(webhookConsumerDurable), nats.ManualAck(),
		nats.MaxDeliver(consumers.MaxDeliverBeforeDeadLetter), nats.BackOff(consumers.BackoffSchedule))
	if err != nil {
		log.Fatalf("subscribe webhooks.received.*: %v", err)
	}
	defer sub.Drain()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Println("webhook-processor started")
	<-ctx.Done()
	log.Println("webhook-processor shutting down")
}
