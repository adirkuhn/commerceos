package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
}

func NewDeleteProductUseCase(db *gorm.DB, productRepo pmsDomain.ProductRepository) *DeleteProductUseCase {
	return &DeleteProductUseCase{db: db, productRepo: productRepo}
}

// Execute deletes the product; variants/skus cascade via the DB's ON DELETE
// CASCADE (migrations/002_pms.sql).
func (uc *DeleteProductUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.productRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.productRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteProduct: %w", err)
	}
	return nil
}
