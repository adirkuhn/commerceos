package pms

import (
	"context"
	"errors"
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
	// Slug, when non-nil, explicitly changes the product's slug -- nil
	// leaves it untouched (the "never re-slug on rename" behavior below
	// still holds for Name changes). This is the only way to fix a
	// product stuck with a duplicate's "-copy" slug suffix.
	Slug *string
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

// Execute updates product fields. Slug is only changed when in.Slug is
// explicitly set -- re-slugging on every Name change would churn the
// product's canonical URL, so a bare rename never touches it.
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
		if in.Slug != nil && *in.Slug != existing.Slug {
			slug := *in.Slug
			if slug == "" || Slugify(slug) != slug {
				return pmsDomain.ErrInvalidSlug
			}
			other, err := uc.productRepo.FindBySlug(ctx, slug)
			if err == nil && other.ID != existing.ID {
				return pmsDomain.ErrDuplicateSlug
			}
			if err != nil && !errors.Is(err, pmsDomain.ErrProductNotFound) {
				return err
			}
			existing.Slug = slug
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
