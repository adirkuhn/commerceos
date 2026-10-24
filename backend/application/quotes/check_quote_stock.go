package quotes

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// stockChecker is inventory's AllocateSaleStockUseCase.CheckAvailability:
// the same FEFO plan a sales order will take, without writing anything.
type stockChecker interface {
	CheckAvailability(ctx context.Context, skuID, warehouseID string, quantity int64) (short int64, err error)
}

// StockCheckLine is one quote line against a warehouse's stock.
type StockCheckLine struct {
	ProductID string
	SKUID     string
	Quantity  int64
	Available int64
	Short     int64
}

type StockCheck struct {
	Lines []StockCheckLine
}

func (c StockCheck) HasShortfall() bool {
	for _, l := range c.Lines {
		if l.Short > 0 {
			return true
		}
	}
	return false
}

// CheckQuoteStockUseCase tells, before converting a quote, which lines a
// warehouse can't fully cover.
type CheckQuoteStockUseCase struct {
	db    *gorm.DB
	repo  quotes.Repository
	stock stockChecker
}

func NewCheckQuoteStockUseCase(db *gorm.DB, repo quotes.Repository, stock stockChecker) *CheckQuoteStockUseCase {
	return &CheckQuoteStockUseCase{db: db, repo: repo, stock: stock}
}

func (uc *CheckQuoteStockUseCase) Execute(ctx context.Context, quoteID, warehouseID string) (*StockCheck, error) {
	q, err := loadQuote(ctx, uc.db, uc.repo, quoteID)
	if err != nil {
		return nil, fmt.Errorf("checkQuoteStock: %w", err)
	}
	check := &StockCheck{}
	for _, l := range q.Lines {
		if l.Quantity <= 0 {
			continue
		}
		short, err := uc.stock.CheckAvailability(ctx, l.SKUID, warehouseID, l.Quantity)
		if err != nil {
			return nil, fmt.Errorf("checkQuoteStock: %w", err)
		}
		check.Lines = append(check.Lines, StockCheckLine{
			ProductID: l.ProductID, SKUID: l.SKUID, Quantity: l.Quantity, Available: l.Quantity - short, Short: short,
		})
	}
	return check, nil
}

func loadQuote(ctx context.Context, db *gorm.DB, repo quotes.Repository, id string) (*quotes.Quote, error) {
	var q *quotes.Quote
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		q, err = repo.FindByID(ctx, id)
		return err
	})
	return q, err
}
