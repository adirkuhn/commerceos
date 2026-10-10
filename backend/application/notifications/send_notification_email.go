package notifications

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/adirkuhn/commerceos/application/tenantemail"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type connectionLister interface {
	ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error)
}

// tenantSettingsGetter is application/settings' GetTenantSettingsUseCase
// narrowed to point of use -- the company language emails are written in.
type tenantSettingsGetter interface {
	Execute(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

// SendNotificationEmailInput: a translated Type is written from Params in
// the company language; Subject/Body are the fallback for other types.
type SendNotificationEmailInput struct {
	Recipients []string
	Type       notificationsDomain.Type
	Params     map[string]any
	Subject    string
	Body       string
	Link       string // frontend-relative, e.g. "/sync-log"
}

// SendNotificationEmailUseCase emails one notification to the tenant's
// chosen recipients, through the tenant's own email connection when it has
// one (see tenantemail.Resolve). Run by the NotificationEmailRequested
// consumer, never inside the producer's transaction.
type SendNotificationEmailUseCase struct {
	db               *gorm.DB
	conns            connectionLister
	encryptionSecret string
	factory          messaging.EmailClientFactory
	fallback         messaging.Client
	appBaseURL       string
	settings         tenantSettingsGetter
}

func NewSendNotificationEmailUseCase(db *gorm.DB, conns connectionLister, encryptionSecret string, factory messaging.EmailClientFactory, fallback messaging.Client, appBaseURL string, settings tenantSettingsGetter) *SendNotificationEmailUseCase {
	return &SendNotificationEmailUseCase{
		db: db, conns: conns, encryptionSecret: encryptionSecret, factory: factory, fallback: fallback,
		appBaseURL: strings.TrimRight(appBaseURL, "/"), settings: settings,
	}
}

func (uc *SendNotificationEmailUseCase) Execute(ctx context.Context, in SendNotificationEmailInput) error {
	s, err := uc.settings.Execute(ctx)
	if err != nil {
		return fmt.Errorf("sendNotificationEmail: %w", err)
	}
	lang := s.Language

	var client messaging.Client
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conns, err := uc.conns.ListByTenant(ctx)
		if err != nil {
			return err
		}
		client, err = tenantemail.Resolve(conns, uc.encryptionSecret, uc.factory, uc.fallback)
		return err
	})
	if err != nil {
		return fmt.Errorf("sendNotificationEmail: %w", err)
	}

	chrome := notificationsDomain.EmailTextsFor(lang)
	text := notificationsDomain.Text{Title: in.Subject, Message: in.Body}
	if translated, ok := in.Type.Render(lang, in.Params); ok {
		text = translated
	}
	body := uc.htmlBody(text, in.Link, chrome)
	for _, to := range in.Recipients {
		msg := messaging.Message{Channel: messaging.ChannelEmail, To: to, Subject: chrome.SubjectPrefix + text.Title, HTMLBody: body, Body: text.Message}
		if err := client.Send(ctx, msg); err != nil {
			return fmt.Errorf("sendNotificationEmail: send to %s: %w", to, err)
		}
	}
	return nil
}

func (uc *SendNotificationEmailUseCase) htmlBody(text notificationsDomain.Text, link string, chrome notificationsDomain.EmailTexts) string {
	var b strings.Builder
	b.WriteString("<p><strong>" + html.EscapeString(text.Title) + "</strong></p>")
	if text.Message != "" {
		b.WriteString("<p>" + html.EscapeString(text.Message) + "</p>")
	}
	if link != "" && uc.appBaseURL != "" {
		b.WriteString(`<p><a href="` + html.EscapeString(uc.appBaseURL+link) + `">` + html.EscapeString(chrome.OpenLink) + `</a></p>`)
	}
	b.WriteString(`<p style="color:#6b7280;font-size:12px">` + html.EscapeString(chrome.Footer) + `</p>`)
	return b.String()
}
