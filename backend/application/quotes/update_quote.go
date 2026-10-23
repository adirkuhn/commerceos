package quotes

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// UpdateQuoteUseCase saves new content over an existing quote; its number
// and author stay.
type UpdateQuoteUseCase struct {
	db     *gorm.DB
	repo   quotes.Repository
	outbox eventrecord.Enqueuer
}

func NewUpdateQuoteUseCase(db *gorm.DB, repo quotes.Repository, outbox eventrecord.Enqueuer) *UpdateQuoteUseCase {
	return &UpdateQuoteUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *UpdateQuoteUseCase) Execute(ctx context.Context, id string, in quotes.Input) (*quotes.Quote, error) {
	var q *quotes.Quote
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if q, err = uc.repo.FindByID(ctx, id); err != nil {
			return err
		}
		before := *q
		if err := q.Revise(in, time.Now().UTC()); err != nil {
			return err
		}
		if err := uc.repo.Update(ctx, q); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.QuoteUpdated, q.ID, eventrecord.NewChanges().
			Set("name", before.Name, q.Name).
			Set("discount_basis_points", before.DiscountBasisPoints, q.DiscountBasisPoints).
			Set("currency_code", before.CurrencyCode, q.CurrencyCode).
			Set("line_count", len(before.Lines), len(q.Lines)).
			Set("total_minor_units", before.Total(), q.Total()))
	})
	if err != nil {
		return nil, fmt.Errorf("updateQuote: %w", err)
	}
	return q, nil
}
