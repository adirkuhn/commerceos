package quotes

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type GetQuoteUseCase struct {
	db   *gorm.DB
	repo quotes.Repository
}

func NewGetQuoteUseCase(db *gorm.DB, repo quotes.Repository) *GetQuoteUseCase {
	return &GetQuoteUseCase{db: db, repo: repo}
}

func (uc *GetQuoteUseCase) Execute(ctx context.Context, id string) (*quotes.Quote, error) {
	var q *quotes.Quote
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		q, err = uc.repo.FindByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getQuote: %w", err)
	}
	return q, nil
}
