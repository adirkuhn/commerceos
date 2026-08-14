package tax

import (
	"context"
	"fmt"
	"strings"
	"time"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateTaxClassInput struct {
	ID        string
	Name      string
	IsDefault bool
}

type UpdateTaxClassUseCase struct {
	db   *gorm.DB
	repo taxDomain.TaxClassRepository
}

func NewUpdateTaxClassUseCase(db *gorm.DB, repo taxDomain.TaxClassRepository) *UpdateTaxClassUseCase {
	return &UpdateTaxClassUseCase{db: db, repo: repo}
}

func (uc *UpdateTaxClassUseCase) Execute(ctx context.Context, in UpdateTaxClassInput) (*taxDomain.TaxClass, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("updateTaxClass: %w", taxDomain.ErrTaxClassNameRequired)
	}

	var class *taxDomain.TaxClass
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if in.IsDefault && !existing.IsDefault {
			if err := unsetExistingDefault(ctx, uc.repo, existing.ID); err != nil {
				return err
			}
		}
		existing.Name = name
		existing.IsDefault = in.IsDefault
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		class = existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateTaxClass: %w", err)
	}
	return class, nil
}
