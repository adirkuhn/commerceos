package moneybird

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// DeleteDocumentUseCase deletes a quote's estimate or an order's invoice
// from Moneybird and drops the link, so it can be sent again -- e.g. an
// order booked with the wrong VAT (a booked external invoice can't be
// changed). The link stays when Moneybird refuses the delete.
type DeleteDocumentUseCase struct {
	db       *gorm.DB
	conns    connectionLister
	settings mb.SettingsRepository
	docs     mb.DocumentRepository
	factory  mb.ClientFactory
	outbox   eventrecord.Enqueuer
	secret   string
}

func NewDeleteDocumentUseCase(db *gorm.DB, conns connectionLister, settings mb.SettingsRepository, docs mb.DocumentRepository,
	factory mb.ClientFactory, outbox eventrecord.Enqueuer, secret string) *DeleteDocumentUseCase {
	return &DeleteDocumentUseCase{db: db, conns: conns, settings: settings, docs: docs, factory: factory, outbox: outbox, secret: secret}
}

func (uc *DeleteDocumentUseCase) Execute(ctx context.Context, sourceType mb.SourceType, sourceID string) error {
	acc, err := loadAccount(ctx, uc.db, uc.conns, uc.settings, uc.secret)
	if err != nil {
		return fmt.Errorf("deleteMoneybirdDocument: %w", err)
	}
	doc, err := documentRecorder{db: uc.db, docs: uc.docs}.find(ctx, sourceType, sourceID)
	if err != nil {
		return fmt.Errorf("deleteMoneybirdDocument: %w", err)
	}
	if doc == nil {
		return fmt.Errorf("deleteMoneybirdDocument: %w", mb.ErrDocumentNotFound)
	}
	if doc.MoneybirdID != "" {
		client := acc.client(uc.factory)
		del := client.DeleteInvoice
		if sourceType == mb.SourceQuote {
			del = client.DeleteEstimate
		}
		if err := del(ctx, doc.MoneybirdID); err != nil {
			return fmt.Errorf("deleteMoneybirdDocument: %w", err)
		}
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.docs.Delete(ctx, sourceType, sourceID); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.MoneybirdDocumentDeleted, sourceID, eventrecord.Data{
			"source_type": string(sourceType), "moneybird_id": doc.MoneybirdID, "reference": doc.Reference,
		})
	})
	if err != nil {
		return fmt.Errorf("deleteMoneybirdDocument: %w", err)
	}
	return nil
}
