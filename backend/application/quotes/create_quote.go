// Package quotes holds the saved B2B quote use cases.
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

type CreateQuoteUseCase struct {
	db     *gorm.DB
	repo   quotes.Repository
	outbox eventrecord.Enqueuer
}

func NewCreateQuoteUseCase(db *gorm.DB, repo quotes.Repository, outbox eventrecord.Enqueuer) *CreateQuoteUseCase {
	return &CreateQuoteUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *CreateQuoteUseCase) Execute(ctx context.Context, in quotes.Input) (*quotes.Quote, error) {
	q, err := quotes.NewQuote(tenant.IDFromContext(ctx), in, tenant.UserIDFromContext(ctx), time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("createQuote: %w", err)
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Create(ctx, q); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.QuoteCreated, q.ID, eventrecord.Data{
			"number": q.Number, "name": q.Name, "line_count": len(q.Lines),
			"total_minor_units": q.Total(), "currency_code": q.CurrencyCode,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createQuote: %w", err)
	}
	return q, nil
}
