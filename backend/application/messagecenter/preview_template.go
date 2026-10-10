package messagecenter

import (
	"context"
	"fmt"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type PreviewTemplateInput struct {
	TriggerSubject string
	Subject        string
	Body           string
	TextBody       string
}

// PreviewTemplate is a template rendered the way it will be sent.
type PreviewTemplate struct {
	Subject string
	HTML    string
	Text    string
}

// PreviewTemplateUseCase renders a (possibly unsaved) template with the
// trigger's sample data and the store's real details -- through the same
// renderer real sends use, so the preview matches the message.
type PreviewTemplateUseCase struct {
	db      *gorm.DB
	profile companyProfileReader
}

func NewPreviewTemplateUseCase(db *gorm.DB, profile companyProfileReader) *PreviewTemplateUseCase {
	return &PreviewTemplateUseCase{db: db, profile: profile}
}

func (uc *PreviewTemplateUseCase) Execute(ctx context.Context, in PreviewTemplateInput) (*PreviewTemplate, error) {
	trigger, ok := mcDomain.TriggerFor(in.TriggerSubject)
	if !ok {
		return nil, fmt.Errorf("previewTemplate: %w", mcDomain.ErrInvalidTriggerSubject)
	}
	data := trigger.SampleData()
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.profile.Get(ctx)
		if err != nil {
			return err
		}
		merge(data, storeVars(p))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("previewTemplate: %w", err)
	}
	tmpl := mcDomain.MessageTemplate{Body: in.Body, TextBody: in.TextBody}
	return &PreviewTemplate{
		Subject: renderSubject(in.Subject, data),
		HTML:    renderBody(tmpl.BodyFor(messaging.ChannelEmail), data, messaging.ChannelEmail),
		Text:    renderBody(tmpl.BodyFor(messaging.ChannelWhatsApp), data, messaging.ChannelWhatsApp),
	}, nil
}
