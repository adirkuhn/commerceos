package notifications

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/adirkuhn/commerceos/application/tenantemail"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type connectionLister interface {
	ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error)
}

type SendNotificationEmailInput struct {
	Recipients []string
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
}

func NewSendNotificationEmailUseCase(db *gorm.DB, conns connectionLister, encryptionSecret string, factory messaging.EmailClientFactory, fallback messaging.Client, appBaseURL string) *SendNotificationEmailUseCase {
	return &SendNotificationEmailUseCase{
		db: db, conns: conns, encryptionSecret: encryptionSecret, factory: factory, fallback: fallback,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
	}
}

func (uc *SendNotificationEmailUseCase) Execute(ctx context.Context, in SendNotificationEmailInput) error {
	var client messaging.Client
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
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

	body := uc.htmlBody(in)
	for _, to := range in.Recipients {
		msg := messaging.Message{Channel: messaging.ChannelEmail, To: to, Subject: "CommerceOS: " + in.Subject, HTMLBody: body, Body: in.Body}
		if err := client.Send(ctx, msg); err != nil {
			return fmt.Errorf("sendNotificationEmail: send to %s: %w", to, err)
		}
	}
	return nil
}

func (uc *SendNotificationEmailUseCase) htmlBody(in SendNotificationEmailInput) string {
	var b strings.Builder
	b.WriteString("<p><strong>" + html.EscapeString(in.Subject) + "</strong></p>")
	if in.Body != "" {
		b.WriteString("<p>" + html.EscapeString(in.Body) + "</p>")
	}
	if in.Link != "" && uc.appBaseURL != "" {
		b.WriteString(`<p><a href="` + html.EscapeString(uc.appBaseURL+in.Link) + `">Open in CommerceOS</a></p>`)
	}
	b.WriteString(`<p style="color:#6b7280;font-size:12px">You get this because of your notification settings in CommerceOS (Settings, Notifications).</p>`)
	return b.String()
}
