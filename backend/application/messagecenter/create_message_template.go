// Package messagecenter holds the merchant-facing CRUD use cases over
// domain/messagecenter.MessageTemplate. Automation (event -> rule ->
// template -> send) is a separate, later slice on top of this one.
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

type CreateMessageTemplateInput struct {
	Name     string
	Category mcDomain.Category
	Channels []messaging.Channel
	Subject  string
	// Body is the email HTML; TextBody the WhatsApp/SMS text; Design the
	// visual editor's project JSON (nil when written by hand).
	Body     string
	TextBody string
	Design   []byte
}

// validateMessageTemplateInput is shared by Create and Update -- same field
// rules apply whichever way a template got into this shape.
func validateMessageTemplateInput(name string, category mcDomain.Category, channels []messaging.Channel, body, textBody string) error {
	if name == "" {
		return mcDomain.ErrTemplateNameRequired
	}
	if !category.Valid() {
		return mcDomain.ErrInvalidCategory
	}
	if len(channels) == 0 {
		return mcDomain.ErrNoChannelsSelected
	}
	for _, ch := range channels {
		switch ch {
		case messaging.ChannelEmail, messaging.ChannelSMS, messaging.ChannelWhatsApp:
		default:
			return mcDomain.ErrInvalidChannel
		}
	}
	return mcDomain.ValidateContent(channels, body, textBody)
}

type CreateMessageTemplateUseCase struct {
	db     *gorm.DB
	repo   mcDomain.MessageTemplateRepository
	outbox eventrecord.Enqueuer
}

func NewCreateMessageTemplateUseCase(db *gorm.DB, repo mcDomain.MessageTemplateRepository, outbox eventrecord.Enqueuer) *CreateMessageTemplateUseCase {
	return &CreateMessageTemplateUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *CreateMessageTemplateUseCase) Execute(ctx context.Context, in CreateMessageTemplateInput) (*mcDomain.MessageTemplate, error) {
	if err := validateMessageTemplateInput(in.Name, in.Category, in.Channels, in.Body, in.TextBody); err != nil {
		return nil, fmt.Errorf("createMessageTemplate: %w", err)
	}

	var tmpl mcDomain.MessageTemplate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		tmpl = mcDomain.MessageTemplate{
			TenantID: tenant.IDFromContext(ctx), Name: in.Name, Category: in.Category,
			Channels: in.Channels, Subject: in.Subject, Body: in.Body, TextBody: in.TextBody, Design: in.Design,
			Enabled: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repo.Create(ctx, &tmpl); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.MessageTemplateCreated, tmpl.ID, eventrecord.Data{
			"name": tmpl.Name, "category": string(tmpl.Category), "channels": tmpl.Channels,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createMessageTemplate: %w", err)
	}
	return &tmpl, nil
}
