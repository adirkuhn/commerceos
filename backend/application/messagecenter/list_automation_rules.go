package messagecenter

import (
	"context"
	"fmt"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListAutomationRulesUseCase struct {
	db   *gorm.DB
	repo mcDomain.AutomationRuleRepository
}

func NewListAutomationRulesUseCase(db *gorm.DB, repo mcDomain.AutomationRuleRepository) *ListAutomationRulesUseCase {
	return &ListAutomationRulesUseCase{db: db, repo: repo}
}

func (uc *ListAutomationRulesUseCase) Execute(ctx context.Context) ([]mcDomain.AutomationRule, error) {
	var rules []mcDomain.AutomationRule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		rules, err = uc.repo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listAutomationRules: %w", err)
	}
	return rules, nil
}
