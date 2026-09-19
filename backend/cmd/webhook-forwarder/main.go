// cmd/webhook-forwarder drains the webhook_forward_deliveries queue --
// separate deployable from cmd/webhook-processor (which only persists +
// dispatches inbound events) and from cmd/relay (which drains the
// domain-event outbox), mirroring this app's one-small-process-per-concern
// convention. Cross-tenant by nature (there's no single tenant driving a
// tick), so it runs every poll under tenant.WithSuperAdmin -- same
// convention as cmd/scheduler's runDue.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/webhookforward"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// pollInterval is short (unlike cmd/relay's own 2s tick this is a bit more
// relaxed since a fan-out target is expected to be a real third-party
// service, not the always-on message broker) -- pending rows are durable in
// Postgres either way, so a slower poll just means a slightly longer worst-
// case delay before the first attempt, never a lost delivery.
const pollInterval = 5 * time.Second

func main() {
	cfg := config.Load()

	db, err := db.Open(cfg.DBDsn, 10)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	deliveryRepo := repositories.NewWebhookForwardDeliveryRepo(db)
	deliverUC := appwebhooks.NewDeliverForwardedWebhooksUseCase(db, deliveryRepo, webhookforward.NewClient())

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	adminCtx := tenant.WithSuperAdmin(ctx)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Println("webhook-forwarder started")
	for {
		select {
		case <-ctx.Done():
			log.Println("webhook-forwarder shutting down")
			return
		case <-ticker.C:
			result, err := deliverUC.Execute(adminCtx)
			if err != nil {
				log.Printf("deliver forwarded webhooks: %v", err)
				continue
			}
			if len(result.Errors) > 0 {
				log.Printf("deliver forwarded webhooks: checked=%d delivered=%d errors=%v",
					result.Checked, result.Delivered, result.Errors)
			}
		}
	}
}
