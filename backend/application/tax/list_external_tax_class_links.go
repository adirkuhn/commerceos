package tax

import (
	"context"
	"fmt"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListExternalTaxClassLinksUseCase struct {
	db   *gorm.DB
	repo taxDomain.ExternalTaxClassLinkRepository
}

func NewListExternalTaxClassLinksUseCase(db *gorm.DB, repo taxDomain.ExternalTaxClassLinkRepository) *ListExternalTaxClassLinksUseCase {
	return &ListExternalTaxClassLinksUseCase{db: db, repo: repo}
}

func (uc *ListExternalTaxClassLinksUseCase) Execute(ctx context.Context) ([]taxDomain.ExternalTaxClassLink, error) {
	var links []taxDomain.ExternalTaxClassLink
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		links, err = uc.repo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listExternalTaxClassLinks: %w", err)
	}
	return links, nil
}
