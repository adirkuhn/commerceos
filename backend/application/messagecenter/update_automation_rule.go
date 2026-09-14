package messagecenter

import (
	"context"
	"fmt"
	"time"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateAutomationRuleInput struct {
	ID              string
	Name            string
	TriggerSubject  string
	Conditions      []mcDomain.Condition
	TemplateID      string
	ChannelOverride *messaging.Channel
	Enabled         bool
}

type UpdateAutomationRuleUseCase struct {
	db           *gorm.DB
	repo         mcDomain.AutomationRuleRepository
	templateRepo mcDomain.MessageTemplateRepository
}

func NewUpdateAutomationRuleUseCase(db *gorm.DB, repo mcDomain.AutomationRuleRepository, templateRepo mcDomain.MessageTemplateRepository) *UpdateAutomationRuleUseCase {
	return &UpdateAutomationRuleUseCase{db: db, repo: repo, templateRepo: templateRepo}
}

func (uc *UpdateAutomationRuleUseCase) Execute(ctx context.Context, in UpdateAutomationRuleInput) (*mcDomain.AutomationRule, error) {
	if err := validateAutomationRuleInput(in.Name, in.TriggerSubject, in.Conditions, in.TemplateID); err != nil {
		return nil, fmt.Errorf("updateAutomationRule: %w", err)
	}

	var rule mcDomain.AutomationRule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if _, err := uc.templateRepo.FindByID(ctx, in.TemplateID); err != nil {
			return err
		}
		existing.Name = in.Name
		existing.TriggerSubject = in.TriggerSubject
		existing.Conditions = in.Conditions
		existing.TemplateID = in.TemplateID
		existing.ChannelOverride = in.ChannelOverride
		existing.Enabled = in.Enabled
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		rule = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateAutomationRule: %w", err)
	}
	return &rule, nil
}
