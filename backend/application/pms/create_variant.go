package pms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateVariantInput struct {
	ProductID string
	Name      string
	Position  int
}

type CreateVariantUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	outboxRepo  outbox.Repository
}

func NewCreateVariantUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository, variantRepo pmsDomain.VariantRepository, outboxRepo outbox.Repository) *CreateVariantUseCase {
	return &CreateVariantUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo, outboxRepo: outboxRepo}
}

func (uc *CreateVariantUseCase) Execute(ctx context.Context, in CreateVariantInput) (*pmsDomain.Variant, error) {
	var variant pmsDomain.Variant
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		product, err := uc.productRepo.FindByID(ctx, in.ProductID)
		if err != nil {
			return err
		}
		if !product.HasVariants {
			existing, err := uc.variantRepo.ListByProduct(ctx, in.ProductID)
			if err != nil {
				return err
			}
			if len(existing) >= 1 {
				return pmsDomain.ErrVariantsNotEnabled
			}
		}

		now := time.Now()
		variant = pmsDomain.Variant{
			TenantID:  tenant.IDFromContext(ctx),
			ProductID: in.ProductID,
			Name:      in.Name,
			Position:  in.Position,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.variantRepo.Create(ctx, &variant); err != nil {
			return err
		}
		// A brand-new variant-mode product's very first enrollment attempt
		// (fired by CreateProductUseCase's own pms.product.changed, right
		// when the product row is created) finds zero variants and bails
		// out without ever creating a ProductChannelListing -- this variant
		// existing now is exactly what that attempt was missing, so it must
		// re-trigger enrollment itself rather than leave the product
		// permanently unsynced until some unrelated later edit happens to.
		return enqueueProductChanged(ctx, uc.outboxRepo, in.ProductID)
	})
	if err != nil {
		return nil, fmt.Errorf("createVariant: %w", err)
	}
	return &variant, nil
}
