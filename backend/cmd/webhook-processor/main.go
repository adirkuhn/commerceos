// cmd/webhook-processor is the cold-path deployable for inbound webhooks --
// separate from cmd/api (which only verifies+publishes) and from
// cmd/relay/cmd/scheduler (which handle the outbox-based domain-event
// path), mirroring this app's existing convention of one small process per
// concern. Multiple instances can run at once: they share a NATS queue
// group, so JetStream load-balances COMMERCEOS_WEBHOOKS across them.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
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

func main() {
	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
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
	recordMovement := appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo)
	allocateStock := appinventory.NewAllocateSaleStockUseCase(db, movementRepo, batchRepo, outboxRepo)
	updateSalesOrder := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, allocateStock, recordMovement)

	shipmentHandler := appshipping.NewHandleShipmentWebhookUseCase(
		db, repositories.NewShipmentRepo(db), repositories.NewShipmentStatusEventRepo(db),
		repositories.NewConnectionRepo(db), salesOrderRepo,
		myparcel.NewClientFactory(), cfg.IntegrationsEncryptionSecret, updateSalesOrder,
	)
	orderStatusHandler := appshipping.NewHandleOrderStatusWebhookUseCase(
		db, repositories.NewMyParcelFulfilmentOrderRepo(db), salesOrderRepo, updateSalesOrder,
	)
	myParcelRouter := appshipping.NewMyParcelWebhookRouter(shipmentHandler, orderStatusHandler)
	forwardRuleRepo := repositories.NewWebhookForwardRuleRepo(db)
	forwardDeliveryRepo := repositories.NewWebhookForwardDeliveryRepo(db)
	processWebhookEventUC := appwebhooks.NewProcessWebhookEventUseCase(db, eventRepo, myParcelRouter, forwardRuleRepo, forwardDeliveryRepo)
	webhookConsumer := consumers.NewWebhookConsumer(processedRepo, processWebhookEventUC, js)

	// "webhooks.received.*" -- one subscription covers every provider
	// (see infrastructure/webhookproviders for the registry cmd/api's hot
	// path verifies against); the consumer itself doesn't branch on
	// provider name.
	sub, err := js.QueueSubscribe("webhooks.received.*", webhookQueueGroup, webhookConsumer.Handle,
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
