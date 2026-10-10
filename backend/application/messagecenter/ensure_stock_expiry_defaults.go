package messagecenter

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// tenantSettingsGetter is application/settings' GetTenantSettingsUseCase
// narrowed to point of use.
type tenantSettingsGetter interface {
	Execute(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

// EnsureStockExpiryDefaultsUseCase gives a tenant the default "stock close
// to expiry" template and rule (mcDomain.DefaultStockExpiry, in the
// tenant's language) when it has no rule for that trigger yet -- run when
// the tenant turns the report on. A tenant that already has such a rule,
// even a disabled or edited one, keeps it untouched.
type EnsureStockExpiryDefaultsUseCase struct {
	db           *gorm.DB
	ruleRepo     mcDomain.AutomationRuleRepository
	templateRepo mcDomain.MessageTemplateRepository
	outbox       eventrecord.Enqueuer
	settings     tenantSettingsGetter
}

func NewEnsureStockExpiryDefaultsUseCase(db *gorm.DB, ruleRepo mcDomain.AutomationRuleRepository, templateRepo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer, settings tenantSettingsGetter) *EnsureStockExpiryDefaultsUseCase {
	return &EnsureStockExpiryDefaultsUseCase{db: db, ruleRepo: ruleRepo, templateRepo: templateRepo, outbox: outbox, settings: settings}
}

func (uc *EnsureStockExpiryDefaultsUseCase) Execute(ctx context.Context) error {
	s, err := uc.settings.Execute(ctx)
	if err != nil {
		return fmt.Errorf("ensureStockExpiryDefaults: %w", err)
	}
	defaults := mcDomain.DefaultStockExpiry(s.Language)

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		rules, err := uc.ruleRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, r := range rules {
			if r.TriggerSubject == mcDomain.TriggerStockCloseToExpiry {
				return nil
			}
		}

		now := time.Now()
		tmpl := defaults.Template
		tmpl.TenantID, tmpl.CreatedAt, tmpl.UpdatedAt = tenant.IDFromContext(ctx), now, now
		if err := uc.templateRepo.Create(ctx, &tmpl); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.MessageTemplateCreated, tmpl.ID, eventrecord.Data{
			"name": tmpl.Name, "category": string(tmpl.Category), "channels": tmpl.Channels,
		}); err != nil {
			return err
		}

		rule := defaults.Rule
		rule.TenantID, rule.TemplateID, rule.ExtraRecipients = tmpl.TenantID, tmpl.ID, []string{}
		rule.CreatedAt, rule.UpdatedAt = now, now
		if err := uc.ruleRepo.Create(ctx, &rule); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AutomationRuleCreated, rule.ID, eventrecord.Data{
			"name": rule.Name, "trigger_subject": rule.TriggerSubject, "template_id": rule.TemplateID,
		})
	})
	if err != nil {
		return fmt.Errorf("ensureStockExpiryDefaults: %w", err)
	}
	return nil
}
