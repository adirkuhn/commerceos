package quotes

import (
	"context"
	"fmt"
	"strings"

	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultPerPage = 50
	maxPerPage     = 200
)

type ListQuotesInput struct {
	Page    int
	PerPage int
	Search  string
}

type ListQuotesResult struct {
	Quotes []quotes.Summary
	Total  int64
}

// ListQuotesUseCase lists saved quotes, most recently saved first.
type ListQuotesUseCase struct {
	db   *gorm.DB
	repo quotes.Repository
}

func NewListQuotesUseCase(db *gorm.DB, repo quotes.Repository) *ListQuotesUseCase {
	return &ListQuotesUseCase{db: db, repo: repo}
}

func (uc *ListQuotesUseCase) Execute(ctx context.Context, in ListQuotesInput) (*ListQuotesResult, error) {
	page := max(in.Page, 1)
	perPage := in.PerPage
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	perPage = min(perPage, maxPerPage)
	filter := quotes.ListFilter{Search: strings.TrimSpace(in.Search), Limit: perPage, Offset: (page - 1) * perPage}

	var result ListQuotesResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		result.Quotes, result.Total, err = uc.repo.List(ctx, filter)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listQuotes: %w", err)
	}
	return &result, nil
}
