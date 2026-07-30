package pms

import (
	"context"
	"errors"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateProductInput struct {
	Name        string
	Description string
	CategoryID  *string
	Status      pmsDomain.ProductStatus
	HasVariants bool
}

type CreateProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
}

func NewCreateProductUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository) *CreateProductUseCase {
	return &CreateProductUseCase{db: db, productRepo: productRepo}
}

func (uc *CreateProductUseCase) Execute(ctx context.Context, in CreateProductInput) (*pmsDomain.Product, error) {
	var product pmsDomain.Product
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		slug := Slugify(in.Name)
		_, err := uc.productRepo.FindBySlug(ctx, slug)
		if err == nil {
			return pmsDomain.ErrDuplicateSlug
		}
		if !errors.Is(err, pmsDomain.ErrProductNotFound) {
			return err
		}

		now := time.Now()
		product = pmsDomain.Product{
			TenantID:    tenant.IDFromContext(ctx),
			CategoryID:  in.CategoryID,
			Name:        in.Name,
			Slug:        slug,
			Description: in.Description,
			Status:      in.Status,
			HasVariants: in.HasVariants,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		return uc.productRepo.Create(ctx, &product)
	})
	if err != nil {
		return nil, fmt.Errorf("createProduct: %w", err)
	}
	return &product, nil
}
