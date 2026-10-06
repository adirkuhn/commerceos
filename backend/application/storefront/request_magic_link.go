package storefront

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/shopperemail"
	"strings"
	"time"

	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/messaging"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const magicLinkTokenTTL = 15 * time.Minute

// RequestMagicLinkInput deliberately has no return URL: where the emailed
// link points is the channel's configuration (channels.SalesChannel.
// SignInLink), never the request's -- a caller-chosen URL would let anyone
// send a victim a genuine sign-in link that delivers the token to them.
type RequestMagicLinkInput struct {
	SalesChannelID string
	Email          string
}

type RequestMagicLinkUseCase struct {
	db             *gorm.DB
	channelRepo    channelsDomain.SalesChannelRepository
	customerRepo   customersDomain.CustomerRepository
	createCustomer customerCreator
	tokenRepo      storefrontDomain.MagicLinkTokenRepository
	emailClient    messaging.Client
	// hostedStorefrontURL is where CommerceOS serves themed storefronts.
	hostedStorefrontURL string
	outbox              eventrecord.Enqueuer
}

func NewRequestMagicLinkUseCase(
	db *gorm.DB,
	channelRepo channelsDomain.SalesChannelRepository,
	customerRepo customersDomain.CustomerRepository,
	createCustomer customerCreator,
	tokenRepo storefrontDomain.MagicLinkTokenRepository,
	emailClient messaging.Client,
	hostedStorefrontURL string,
	outbox eventrecord.Enqueuer,
) *RequestMagicLinkUseCase {
	return &RequestMagicLinkUseCase{
		db: db, channelRepo: channelRepo, customerRepo: customerRepo,
		createCustomer: createCustomer, tokenRepo: tokenRepo, emailClient: emailClient,
		hostedStorefrontURL: hostedStorefrontURL, outbox: outbox,
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
	var channel *channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		ch, err := findInternalChannel(ctx, uc.channelRepo, in.SalesChannelID)
		if err != nil {
			return err
		}
		// Fail before creating a customer or token when there is no page
		// to send the shopper to.
		if _, err := ch.SignInLink(uc.hostedStorefrontURL, ""); err != nil {
			return err
		}
		channel = ch
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
		// The shopper's email and the link token stay out of the event.
		if err := eventrecord.Record(ctx, uc.outbox, events.MagicLinkRequested, token.ID, eventrecord.Data{
			"sales_channel_id": token.SalesChannelID, "customer_id": token.CustomerID,
		}); err != nil {
			return err
		}
		signIn, err := channel.SignInLink(uc.hostedStorefrontURL, token.Token)
		link = signIn
		return err
	})
	if err != nil {
		return fmt.Errorf("requestMagicLink: %w", err)
	}

	// Sending email is external I/O -- deliberately outside every
	// transaction above, same "never do external I/O inside
	// tenant.WithTenant" rule this codebase applies to NATS publishes.
	subject, body := shopperemail.SignIn(channel.StorefrontLocale, link)
	msg := messaging.Message{Channel: messaging.ChannelEmail, To: email, Subject: subject, HTMLBody: body}
	if err := uc.emailClient.Send(ctx, msg); err != nil {
		return fmt.Errorf("requestMagicLink: %w", err)
	}
	return nil
}
