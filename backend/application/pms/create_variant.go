package pms

import (
	"context"
	"fmt"
	"time"

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
}

func NewCreateVariantUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository, variantRepo pmsDomain.VariantRepository) *CreateVariantUseCase {
	return &CreateVariantUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo}
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
		return uc.variantRepo.Create(ctx, &variant)
	})
	if err != nil {
		return nil, fmt.Errorf("createVariant: %w", err)
	}
	return &variant, nil
}
