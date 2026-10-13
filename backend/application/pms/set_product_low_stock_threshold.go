package pms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// productLowStockThresholdRepo is ProductRepository narrowed to what
// setting the threshold needs, plus the threshold-only write.
type productLowStockThresholdRepo interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
	SetLowStockThreshold(ctx context.Context, productID string, threshold *int) error
}

// SetProductLowStockThresholdUseCase sets or clears (nil) a product's own
// low stock line. Separate from UpdateProductUseCase so the many product
// updates (admin form, channel imports) never touch it.
type SetProductLowStockThresholdUseCase struct {
	db     *gorm.DB
	repo   productLowStockThresholdRepo
	outbox eventrecord.Enqueuer
}

func NewSetProductLowStockThresholdUseCase(db *gorm.DB, repo productLowStockThresholdRepo, outbox eventrecord.Enqueuer) *SetProductLowStockThresholdUseCase {
	return &SetProductLowStockThresholdUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *SetProductLowStockThresholdUseCase) Execute(ctx context.Context, productID string, threshold *int) (*pmsDomain.Product, error) {
	if err := pmsDomain.ValidateLowStockThreshold(threshold); err != nil {
		return nil, fmt.Errorf("setProductLowStockThreshold: %w", err)
	}
	var product *pmsDomain.Product
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		before, err := uc.repo.FindByID(ctx, productID)
		if err != nil {
			return err
		}
		if err := uc.repo.SetLowStockThreshold(ctx, productID, threshold); err != nil {
			return err
		}
		if product, err = uc.repo.FindByID(ctx, productID); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ProductLowStockThresholdChanged, productID, eventrecord.NewChanges().
			Set("low_stock_threshold", before.LowStockThreshold, product.LowStockThreshold))
	})
	if err != nil {
		return nil, fmt.Errorf("setProductLowStockThreshold: %w", err)
	}
	return product, nil
}
