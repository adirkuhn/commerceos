package quotes

import (
	"context"
	"time"
)

// Summary is a quote as listed: totals without its lines.
type Summary struct {
	ID                  string
	Number              int64
	Name                string
	Customer            *CustomerRef
	CurrencyCode        string
	DiscountBasisPoints int
	LineCount           int
	TotalMinorUnits     int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ListFilter struct {
	// Search matches the name or the reference (Q-0007 or just 7).
	Search string
	Limit  int
	Offset int
}

// Repository stores quotes. Create assigns ID and the next Number for the
// tenant; numbers are never reused, even after a delete.
type Repository interface {
	Create(ctx context.Context, q *Quote) error
	FindByID(ctx context.Context, id string) (*Quote, error)
	List(ctx context.Context, filter ListFilter) ([]Summary, int64, error)
	// Update saves name, customer, discount, currency, lines and UpdatedAt.
	Update(ctx context.Context, q *Quote) error
	Delete(ctx context.Context, id string) error
}
