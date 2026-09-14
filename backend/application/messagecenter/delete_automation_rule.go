package messagecenter

import (
	"context"
	"fmt"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteAutomationRuleUseCase struct {
	db   *gorm.DB
	repo mcDomain.AutomationRuleRepository
}

func NewDeleteAutomationRuleUseCase(db *gorm.DB, repo mcDomain.AutomationRuleRepository) *DeleteAutomationRuleUseCase {
	return &DeleteAutomationRuleUseCase{db: db, repo: repo}
}

func (uc *DeleteAutomationRuleUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.repo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.repo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteAutomationRule: %w", err)
	}
	return nil
}
