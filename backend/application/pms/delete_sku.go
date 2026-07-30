package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteSKUUseCase struct {
	db      *gorm.DB
	skuRepo pmsDomain.SKURepository
}

func NewDeleteSKUUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository) *DeleteSKUUseCase {
	return &DeleteSKUUseCase{db: db, skuRepo: skuRepo}
}

func (uc *DeleteSKUUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.skuRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.skuRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteSKU: %w", err)
	}
	return nil
}
