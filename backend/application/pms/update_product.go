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

type UpdateProductInput struct {
	ID                          string
	Name                        string
	Description                 string
	CategoryID                  *string
	Status                      pmsDomain.ProductStatus
	HasVariants                 bool
	SyncVariantsAsWooVariations bool
	Tags                        []string
}

type UpdateProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	outboxRepo  outbox.Repository
}

func NewUpdateProductUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository, variantRepo pmsDomain.VariantRepository, outboxRepo outbox.Repository) *UpdateProductUseCase {
	return &UpdateProductUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo, outboxRepo: outboxRepo}
}

// Execute updates product fields. Slug is deliberately left unchanged even
// when Name changes — re-slugging on every rename would churn the product's
// canonical URL; slug is only ever generated once, at creation.
func (uc *UpdateProductUseCase) Execute(ctx context.Context, in UpdateProductInput) (*pmsDomain.Product, error) {
	var product pmsDomain.Product
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.productRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if !in.HasVariants {
			variants, err := uc.variantRepo.ListByProduct(ctx, in.ID)
			if err != nil {
				return err
			}
			if len(variants) > 1 {
				return pmsDomain.ErrVariantsNotEnabled
			}
		}
		existing.Name = in.Name
		existing.Description = in.Description
		existing.CategoryID = in.CategoryID
		existing.Status = in.Status
		existing.HasVariants = in.HasVariants
		existing.SyncVariantsAsWooVariations = in.SyncVariantsAsWooVariations
		existing.Tags = in.Tags
		existing.UpdatedAt = time.Now()
		if err := uc.productRepo.Update(ctx, existing); err != nil {
			return err
		}
		product = *existing
		return enqueueProductChanged(ctx, uc.outboxRepo, product.ID)
	})
	if err != nil {
		return nil, fmt.Errorf("updateProduct: %w", err)
	}
	return &product, nil
}
