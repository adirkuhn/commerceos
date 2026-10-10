package messagecenter

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListMessageTemplateDefaultsUseCase lists the built-in default templates
// in the tenant's language -- the "start from a default" gallery.
type ListMessageTemplateDefaultsUseCase struct {
	settings tenantSettingsGetter
}

func NewListMessageTemplateDefaultsUseCase(settings tenantSettingsGetter) *ListMessageTemplateDefaultsUseCase {
	return &ListMessageTemplateDefaultsUseCase{settings: settings}
}

func (uc *ListMessageTemplateDefaultsUseCase) Execute(ctx context.Context) ([]mcDomain.DefaultTemplate, error) {
	s, err := uc.settings.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("listMessageTemplateDefaults: %w", err)
	}
	return mcDomain.DefaultTemplates(s.Language), nil
}

// ResetMessageTemplateUseCase puts a template made from a default back to
// that default's content (subject, email, text) in the tenant's language.
// The visual design is dropped so the editor reopens from the default's
// HTML; name, channels and enabled stay as the tenant set them.
type ResetMessageTemplateUseCase struct {
	db       *gorm.DB
	repo     mcDomain.MessageTemplateRepository
	outbox   eventrecord.Enqueuer
	settings tenantSettingsGetter
}

func NewResetMessageTemplateUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer, settings tenantSettingsGetter) *ResetMessageTemplateUseCase {
	return &ResetMessageTemplateUseCase{db: db, repo: repo, outbox: outbox, settings: settings}
}

func (uc *ResetMessageTemplateUseCase) Execute(ctx context.Context, id string) (*mcDomain.MessageTemplate, error) {
	s, err := uc.settings.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("resetMessageTemplate: %w", err)
	}
	var tmpl mcDomain.MessageTemplate
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if existing.DefaultKey == "" {
			return mcDomain.ErrTemplateHasNoDefault
		}
		def, ok := mcDomain.DefaultTemplateFor(existing.DefaultKey, s.Language)
		if !ok {
			return mcDomain.ErrUnknownDefaultTemplate
		}
		before := *existing
		existing.Subject, existing.Body, existing.TextBody = def.Template.Subject, def.Template.Body, def.Template.TextBody
		existing.Design = nil
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		tmpl = *existing
		return eventrecord.Record(ctx, uc.outbox, events.MessageTemplateUpdated, tmpl.ID, eventrecord.NewChanges().
			Set("subject", before.Subject, tmpl.Subject).
			Set("body", before.Body, tmpl.Body).
			Set("text_body", before.TextBody, tmpl.TextBody).
			Set("has_design", len(before.Design) > 0, false).
			Set("reset_to_default", "", tmpl.DefaultKey))
	})
	if err != nil {
		return nil, fmt.Errorf("resetMessageTemplate: %w", err)
	}
	return &tmpl, nil
}
