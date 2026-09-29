package newsletter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
	"github.com/adirkuhn/commerceos/application/tenantemail"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/smtp"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// tenantMailer sends as the current tenant: its own email connection when
// it has one, otherwise the platform SMTP client.
type tenantMailer struct {
	db       *gorm.DB
	conns    *repositories.ConnectionRepo
	secret   string
	fallback messaging.Client
}

func (m tenantMailer) Send(ctx context.Context, msg messaging.Message) error {
	var client messaging.Client
	err := tenant.WithTenant(ctx, m.db, func(ctx context.Context) error {
		conns, err := m.conns.ListByTenant(ctx)
		if err != nil {
			return err
		}
		client, err = tenantemail.Resolve(conns, m.secret, smtp.NewClientFactory(), m.fallback)
		return err
	})
	if err != nil {
		return fmt.Errorf("newsletter mailer: %w", err)
	}
	return client.Send(ctx, msg)
}

// channelStores describes a store by its sales channel: name and shopper
// language (called inside a tenant transaction).
type channelStores struct {
	channels *repositories.SalesChannelRepo
}

func (n channelStores) Store(ctx context.Context, salesChannelID string) (appnewsletter.Store, error) {
	ch, err := n.channels.FindByID(ctx, salesChannelID)
	if err != nil {
		return appnewsletter.Store{}, err
	}
	return appnewsletter.Store{Name: ch.Name, Locale: ch.StorefrontLocale}, nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NewDeps wires the newsletter use cases on the real adapters.
// storefrontURL is where shoppers open the storefront (links in emails).
func NewDeps(db *gorm.DB, encryptionSecret string, platformEmail messaging.Client, storefrontURL string) appnewsletter.Deps {
	base := strings.TrimRight(storefrontURL, "/")
	link := func(page string) func(channelID, token string) string {
		return func(channelID, token string) string {
			return fmt.Sprintf("%s/store/%s/newsletter/%s?token=%s", base, url.PathEscape(channelID), page, url.QueryEscape(token))
		}
	}
	return appnewsletter.Deps{
		DB: db, Repo: repositories.NewNewsletterRepo(db), Outbox: repositories.NewOutboxRepository(db),
		Mailer: tenantMailer{db: db, conns: repositories.NewConnectionRepo(db), secret: encryptionSecret, fallback: platformEmail},
		Stores: channelStores{channels: repositories.NewSalesChannelRepo(db)},
		Now:    time.Now, NewToken: newToken,
		ConfirmURL: link("confirm"), UnsubscribeURL: link("unsubscribe"),
	}
}
