package quotes

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteQuoteUseCase struct {
	db     *gorm.DB
	repo   quotes.Repository
	outbox eventrecord.Enqueuer
}

func NewDeleteQuoteUseCase(db *gorm.DB, repo quotes.Repository, outbox eventrecord.Enqueuer) *DeleteQuoteUseCase {
	return &DeleteQuoteUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteQuoteUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		q, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.QuoteDeleted, id, eventrecord.Data{"number": q.Number, "name": q.Name})
	})
	if err != nil {
		return fmt.Errorf("deleteQuote: %w", err)
	}
	return nil
}
