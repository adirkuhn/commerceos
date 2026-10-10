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

type UpdateMessageTemplateInput struct {
	ID       string
	Name     string
	Category mcDomain.Category
	Channels []messaging.Channel
	Subject  string
	Body     string
	TextBody string
	// Design replaces the visual editor's project JSON when set; nil keeps
	// it (the simple template form never sends it).
	Design  *[]byte
	Enabled bool
}

type UpdateMessageTemplateUseCase struct {
	db     *gorm.DB
	repo   mcDomain.MessageTemplateRepository
	outbox eventrecord.Enqueuer
}

func NewUpdateMessageTemplateUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer) *UpdateMessageTemplateUseCase {
	return &UpdateMessageTemplateUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *UpdateMessageTemplateUseCase) Execute(ctx context.Context, in UpdateMessageTemplateInput) (*mcDomain.MessageTemplate, error) {
	if err := validateMessageTemplateInput(in.Name, in.Category, in.Channels, in.Body, in.TextBody); err != nil {
		return nil, fmt.Errorf("updateMessageTemplate: %w", err)
	}

	var tmpl mcDomain.MessageTemplate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		before := *existing
		existing.Name = in.Name
		existing.Category = in.Category
		existing.Channels = in.Channels
		existing.Subject = in.Subject
		existing.Body = in.Body
		existing.TextBody = in.TextBody
		if in.Design != nil {
			existing.Design = *in.Design
		}
		existing.Enabled = in.Enabled
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		tmpl = *existing
		return eventrecord.Record(ctx, uc.outbox, events.MessageTemplateUpdated, tmpl.ID, eventrecord.NewChanges().
			Set("name", before.Name, tmpl.Name).
			Set("category", string(before.Category), string(tmpl.Category)).
			Set("channels", before.Channels, tmpl.Channels).
			Set("subject", before.Subject, tmpl.Subject).
			Set("body", before.Body, tmpl.Body).
			Set("text_body", before.TextBody, tmpl.TextBody).
			Set("has_design", len(before.Design) > 0, len(tmpl.Design) > 0).
			Set("enabled", before.Enabled, tmpl.Enabled))
	})
	if err != nil {
		return nil, fmt.Errorf("updateMessageTemplate: %w", err)
	}
	return &tmpl, nil
}
