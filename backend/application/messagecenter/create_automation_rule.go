package messagecenter

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateAutomationRuleInput struct {
	Name            string
	TriggerSubject  string
	Conditions      []mcDomain.Condition
	TemplateID      string
	ChannelOverride *messaging.Channel
}

// validateAutomationRuleInput is shared by Create and Update -- same field
// rules apply whichever way a rule got into this shape. templateExists is
// injected rather than a repo field so the same validation runs whether or
// not a lookup already happened.
func validateAutomationRuleInput(name, triggerSubject string, conditions []mcDomain.Condition, templateID string) error {
	if name == "" {
		return mcDomain.ErrRuleNameRequired
	}
	if !mcDomain.ValidTriggerSubject(triggerSubject) {
		return mcDomain.ErrInvalidTriggerSubject
	}
	if templateID == "" {
		return mcDomain.ErrTemplateIDRequired
	}
	for _, c := range conditions {
		if c.Field == "" || !c.Operator.Valid() {
			return mcDomain.ErrInvalidCondition
		}
		if c.Operator == mcDomain.OperatorIn {
			if _, ok := c.Value.([]any); !ok {
				return mcDomain.ErrInvalidCondition
			}
		}
	}
	return nil
}

type CreateAutomationRuleUseCase struct {
	db           *gorm.DB
	repo         mcDomain.AutomationRuleRepository
	templateRepo mcDomain.MessageTemplateRepository
	outbox eventrecord.Enqueuer
}

func NewCreateAutomationRuleUseCase(db *gorm.DB, repo mcDomain.AutomationRuleRepository, templateRepo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer) *CreateAutomationRuleUseCase {
	return &CreateAutomationRuleUseCase{db: db, repo: repo, templateRepo: templateRepo, outbox: outbox}
}

func (uc *CreateAutomationRuleUseCase) Execute(ctx context.Context, in CreateAutomationRuleInput) (*mcDomain.AutomationRule, error) {
	if err := validateAutomationRuleInput(in.Name, in.TriggerSubject, in.Conditions, in.TemplateID); err != nil {
		return nil, fmt.Errorf("createAutomationRule: %w", err)
	}

	var rule mcDomain.AutomationRule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.templateRepo.FindByID(ctx, in.TemplateID); err != nil {
			return err
		}
		now := time.Now()
		rule = mcDomain.AutomationRule{
			TenantID: tenant.IDFromContext(ctx), Name: in.Name, TriggerSubject: in.TriggerSubject,
			Conditions: in.Conditions, TemplateID: in.TemplateID, ChannelOverride: in.ChannelOverride,
			Enabled: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Create(ctx, &rule); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AutomationRuleCreated, rule.ID, eventrecord.Data{
			"name": rule.Name, "trigger_subject": rule.TriggerSubject, "template_id": rule.TemplateID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createAutomationRule: %w", err)
	}
	return &rule, nil
}
