package messagecenter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// EnsureMessageCenterDefaultsUseCase gives the tenant in ctx every default
// template with its rule (mcDomain.DefaultTemplates, in the tenant's
// language), all enabled, so the tenant only switches off what it doesn't
// want. It runs once per tenant (SetupRepository); a trigger the tenant
// already has a rule for is left alone.
type EnsureMessageCenterDefaultsUseCase struct {
	db           *gorm.DB
	templateRepo mcDomain.MessageTemplateRepository
	ruleRepo     mcDomain.AutomationRuleRepository
	setup        mcDomain.SetupRepository
	outbox       eventrecord.Enqueuer
	settings     tenantSettingsGetter
}

func NewEnsureMessageCenterDefaultsUseCase(db *gorm.DB, templateRepo mcDomain.MessageTemplateRepository, ruleRepo mcDomain.AutomationRuleRepository,
	setup mcDomain.SetupRepository, outbox eventrecord.Enqueuer, settings tenantSettingsGetter) *EnsureMessageCenterDefaultsUseCase {
	return &EnsureMessageCenterDefaultsUseCase{db: db, templateRepo: templateRepo, ruleRepo: ruleRepo, setup: setup, outbox: outbox, settings: settings}
}

func (uc *EnsureMessageCenterDefaultsUseCase) Execute(ctx context.Context) error {
	if tenant.IDFromContext(ctx) == "" {
		return nil
	}
	s, err := uc.settings.Execute(ctx)
	if err != nil {
		return fmt.Errorf("ensureMessageCenterDefaults: %w", err)
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		claimed, err := uc.setup.Claim(ctx, now)
		if err != nil || !claimed {
			return err
		}
		rules, err := uc.ruleRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		handled := make(map[string]bool, len(rules))
		for _, r := range rules {
			handled[r.TriggerSubject] = true
		}
		for _, d := range mcDomain.DefaultTemplates(s.Language) {
			if handled[d.TriggerSubject] {
				continue
			}
			if err := uc.create(ctx, d, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ensureMessageCenterDefaults: %w", err)
	}
	return nil
}

func (uc *EnsureMessageCenterDefaultsUseCase) create(ctx context.Context, d mcDomain.DefaultTemplate, now time.Time) error {
	tmpl := d.Template
	tmpl.TenantID, tmpl.CreatedAt, tmpl.UpdatedAt = tenant.IDFromContext(ctx), now, now
	if err := uc.templateRepo.Create(ctx, &tmpl); err != nil {
		return err
	}
	if err := eventrecord.Record(ctx, uc.outbox, events.MessageTemplateCreated, tmpl.ID, eventrecord.Data{
		"name": tmpl.Name, "category": string(tmpl.Category), "channels": tmpl.Channels, "default_key": tmpl.DefaultKey,
	}); err != nil {
		return err
	}
	rule := d.Rule
	rule.TenantID, rule.TemplateID, rule.CreatedAt, rule.UpdatedAt = tmpl.TenantID, tmpl.ID, now, now
	if err := uc.ruleRepo.Create(ctx, &rule); err != nil {
		return err
	}
	return eventrecord.Record(ctx, uc.outbox, events.AutomationRuleCreated, rule.ID, eventrecord.Data{
		"name": rule.Name, "trigger_subject": rule.TriggerSubject, "template_id": rule.TemplateID,
	})
}

// defaultsEnsurer is EnsureMessageCenterDefaultsUseCase narrowed to point
// of use.
type defaultsEnsurer interface {
	Execute(ctx context.Context) error
}

// SeedMessageCenterDefaultsUseCase gives every tenant that hasn't had them
// yet the default templates and rules -- run when the scheduler starts, so
// existing tenants get them without opening the Message Center. A failing
// tenant doesn't stop the others.
type SeedMessageCenterDefaultsUseCase struct {
	db      *gorm.DB
	setup   mcDomain.SetupRepository
	ensurer defaultsEnsurer
}

func NewSeedMessageCenterDefaultsUseCase(db *gorm.DB, setup mcDomain.SetupRepository, ensurer defaultsEnsurer) *SeedMessageCenterDefaultsUseCase {
	return &SeedMessageCenterDefaultsUseCase{db: db, setup: setup, ensurer: ensurer}
}

func (uc *SeedMessageCenterDefaultsUseCase) Execute(ctx context.Context) error {
	var tenants []string
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		var err error
		tenants, err = uc.setup.ListUnseededTenants(ctx)
		return err
	})
	if err != nil {
		return fmt.Errorf("seedMessageCenterDefaults: %w", err)
	}
	var failures []error
	for _, id := range tenants {
		if err := uc.ensurer.Execute(tenant.AsTenant(ctx, id)); err != nil {
			failures = append(failures, fmt.Errorf("tenant %s: %w", id, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("seedMessageCenterDefaults: %w", errors.Join(failures...))
	}
	return nil
}

// templateLister is ListMessageTemplatesUseCase narrowed to point of use.
type templateLister interface {
	Execute(ctx context.Context) ([]mcDomain.MessageTemplate, error)
}

// ListMessageTemplatesWithDefaultsUseCase lists the tenant's templates
// after making sure it got the defaults -- how a brand-new tenant gets
// them the first time it opens the Message Center.
type ListMessageTemplatesWithDefaultsUseCase struct {
	ensurer defaultsEnsurer
	list    templateLister
}

func NewListMessageTemplatesWithDefaultsUseCase(ensurer defaultsEnsurer, list templateLister) *ListMessageTemplatesWithDefaultsUseCase {
	return &ListMessageTemplatesWithDefaultsUseCase{ensurer: ensurer, list: list}
}

func (uc *ListMessageTemplatesWithDefaultsUseCase) Execute(ctx context.Context) ([]mcDomain.MessageTemplate, error) {
	if err := uc.ensurer.Execute(ctx); err != nil {
		return nil, fmt.Errorf("listMessageTemplates: %w", err)
	}
	return uc.list.Execute(ctx)
}
