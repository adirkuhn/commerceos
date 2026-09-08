package storefront

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	emailDomain "github.com/adirkuhn/commerceos/domain/email"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const magicLinkTokenTTL = 15 * time.Minute

type RequestMagicLinkInput struct {
	SalesChannelID string
	Email          string
	// RedirectURL is the storefront-frontend's own page that completes
	// sign-in -- the backend has no per-channel frontend URL configured
	// (a tenant's storefronts can live on arbitrary custom domains), so
	// the frontend tells it where to send the shopper back to, same as
	// any third-party magic-link provider's redirectTo parameter. The
	// emailed link is exactly RedirectURL with "?token=..." appended.
	RedirectURL string
}

type RequestMagicLinkUseCase struct {
	db             *gorm.DB
	channelRepo    channelsDomain.SalesChannelRepository
	customerRepo   customersDomain.CustomerRepository
	createCustomer customerCreator
	tokenRepo      storefrontDomain.MagicLinkTokenRepository
	emailClient    emailDomain.Client
}

func NewRequestMagicLinkUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	customerRepo customersDomain.CustomerRepository,
	createCustomer customerCreator,
	tokenRepo storefrontDomain.MagicLinkTokenRepository,
	emailClient emailDomain.Client,
) *RequestMagicLinkUseCase {
	return &RequestMagicLinkUseCase{
		db: db, channelRepo: channelRepo, customerRepo: customerRepo,
		createCustomer: createCustomer, tokenRepo: tokenRepo, emailClient: emailClient,
	}
}

// Execute always finds-or-creates a Customer by email (same convention
// CheckoutCartUseCase.resolveCustomer already uses) and always sends an
// email -- there is no branch where the response or behavior differs
// based on whether the email was already a known customer, so this can
// never be used to check whether an address has an account here.
func (uc *RequestMagicLinkUseCase) Execute(ctx context.Context, in RequestMagicLinkInput) error {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	var existing *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := findInternalChannel(ctx, uc.channelRepo, in.SalesChannelID); err != nil {
			return err
		}
		found, err := uc.customerRepo.FindByEmail(ctx, email)
		if err != nil {
			return err
		}
		existing = found
		return nil
	})
	if err != nil {
		return fmt.Errorf("requestMagicLink: %w", err)
	}

	customerID := ""
	if existing != nil {
		customerID = existing.ID
	} else {
		created, err := uc.createCustomer.Execute(ctx, appcustomers.CreateCustomerInput{Email: email})
		if err != nil {
			return fmt.Errorf("requestMagicLink: %w", err)
		}
		customerID = created.ID
	}

	var link string
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		token := &storefrontDomain.MagicLinkToken{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: in.SalesChannelID, CustomerID: customerID,
			Token: idgen.NewV4(), ExpiresAt: now.Add(magicLinkTokenTTL), CreatedAt: now,
		}
		if err := uc.tokenRepo.Create(ctx, token); err != nil {
			return err
		}
		link = fmt.Sprintf("%s?token=%s", in.RedirectURL, token.Token)
		return nil
	})
	if err != nil {
		return fmt.Errorf("requestMagicLink: %w", err)
	}

	// Sending email is external I/O -- deliberately outside every
	// transaction above, same "never do external I/O inside
	// tenant.WithTenant" rule this codebase applies to NATS publishes.
	msg := emailDomain.Message{
		To:      email,
		Subject: "Your sign-in link",
		HTMLBody: fmt.Sprintf(
			`<p>Click the link below to sign in. It expires in 15 minutes and can only be used once.</p><p><a href="%s">Sign in</a></p>`,
			link,
		),
	}
	if err := uc.emailClient.Send(ctx, msg); err != nil {
		return fmt.Errorf("requestMagicLink: %w", err)
	}
	return nil
}
