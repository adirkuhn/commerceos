package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteExternalTaxClassLinkUseCase struct {
	db   *gorm.DB
	repo taxDomain.ExternalTaxClassLinkRepository
}

func NewDeleteExternalTaxClassLinkUseCase(db *gorm.DB, repo taxDomain.ExternalTaxClassLinkRepository) *DeleteExternalTaxClassLinkUseCase {
	return &DeleteExternalTaxClassLinkUseCase{db: db, repo: repo}
}

func (uc *DeleteExternalTaxClassLinkUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.repo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteExternalTaxClassLink: %w", err)
	}
	return nil
}
