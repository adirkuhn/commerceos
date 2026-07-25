package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
)

func main() {
	cfg := config.Load()

	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Drain()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	repo := repositories.NewOutboxRepository(db)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("relay started")
	for {
		select {
		case <-ctx.Done():
			log.Println("relay shutting down")
			return
		case <-ticker.C:
			publish(ctx, repo, js)
		}
	}
}

func publish(ctx context.Context, repo outbox.Repository, js nats.JetStreamContext) {
	msgs, err := repo.FetchPending(ctx, 50)
	if err != nil {
		log.Printf("fetch pending: %v", err)
		return
	}
	for _, m := range msgs {
		if _, err := js.Publish(m.Subject, m.Payload); err != nil {
			log.Printf("publish %s: %v", m.EventID, err)
			_ = repo.MarkFailed(ctx, m.ID)
			continue
		}
		_ = repo.MarkPublished(ctx, m.ID)
	}
}
