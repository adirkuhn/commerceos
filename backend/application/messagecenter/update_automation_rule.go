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

type UpdateAutomationRuleInput struct {
	ID              string
	Name            string
	TriggerSubject  string
	Conditions      []mcDomain.Condition
	TemplateID      string
	ChannelOverride *messaging.Channel
	ExtraRecipients []string
	Enabled         bool
}

type UpdateAutomationRuleUseCase struct {
	db           *gorm.DB
	repo         mcDomain.AutomationRuleRepository
	templateRepo mcDomain.MessageTemplateRepository
	outbox       eventrecord.Enqueuer
}

func NewUpdateAutomationRuleUseCase(db *gorm.DB, repo mcDomain.AutomationRuleRepository, templateRepo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer) *UpdateAutomationRuleUseCase {
	return &UpdateAutomationRuleUseCase{db: db, repo: repo, templateRepo: templateRepo, outbox: outbox}
}

func (uc *UpdateAutomationRuleUseCase) Execute(ctx context.Context, in UpdateAutomationRuleInput) (*mcDomain.AutomationRule, error) {
	if err := validateAutomationRuleInput(in.Name, in.TriggerSubject, in.Conditions, in.TemplateID); err != nil {
		return nil, fmt.Errorf("updateAutomationRule: %w", err)
	}
	extraRecipients, err := mcDomain.NormalizeExtraRecipients(in.ExtraRecipients)
	if err != nil {
		return nil, fmt.Errorf("updateAutomationRule: %w", err)
	}

	var rule mcDomain.AutomationRule
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if _, err := uc.templateRepo.FindByID(ctx, in.TemplateID); err != nil {
			return err
		}
		before := *existing
		existing.Name = in.Name
		existing.TriggerSubject = in.TriggerSubject
		existing.Conditions = in.Conditions
		existing.TemplateID = in.TemplateID
		existing.ChannelOverride = in.ChannelOverride
		existing.ExtraRecipients = extraRecipients
		existing.Enabled = in.Enabled
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		rule = *existing
		return eventrecord.Record(ctx, uc.outbox, events.AutomationRuleUpdated, rule.ID, eventrecord.NewChanges().
			Set("name", before.Name, rule.Name).
			Set("trigger_subject", before.TriggerSubject, rule.TriggerSubject).
			Set("conditions", before.Conditions, rule.Conditions).
			Set("template_id", before.TemplateID, rule.TemplateID).
			Set("channel_override", before.ChannelOverride, rule.ChannelOverride).
			Set("extra_recipients", nonNilStrings(before.ExtraRecipients), rule.ExtraRecipients).
			Set("enabled", before.Enabled, rule.Enabled))
	})
	if err != nil {
		return nil, fmt.Errorf("updateAutomationRule: %w", err)
	}
	return &rule, nil
}

// nonNilStrings makes a rule saved before extra recipients existed (nil)
// compare equal to an explicitly empty list.
func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
