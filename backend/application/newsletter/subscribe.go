package newsletter

import (
	"context"
	"fmt"
	"html"

	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type SubscribeInput struct {
	SalesChannelID string
	Email          string
	Consent        bool
}

// SubscribeUseCase records a signup and emails a confirmation link. It
// behaves the same whether or not the address is already known (no way to
// probe who's subscribed); repeat signups are throttled per address.
type SubscribeUseCase struct{ deps Deps }

func NewSubscribeUseCase(deps Deps) *SubscribeUseCase { return &SubscribeUseCase{deps: deps} }

func (uc *SubscribeUseCase) Execute(ctx context.Context, in SubscribeInput) error {
	if !in.Consent {
		return fmt.Errorf("subscribe: %w", newsletter.ErrConsentRequired)
	}
	email, err := newsletter.NormalizeEmail(in.Email)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	token, err := uc.deps.NewToken()
	if err != nil {
		return fmt.Errorf("subscribe: token: %w", err)
	}
	var send *newsletter.Subscriber
	var storeName string
	err = tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		now := uc.deps.Now()
		existing, err := uc.deps.Repo.FindByEmail(ctx, email)
		if err != nil {
			return err
		}
		if existing == nil {
			unsub, err := uc.deps.NewToken()
			if err != nil {
				return err
			}
			s := newsletter.NewSubscriber(tenant.IDFromContext(ctx), in.SalesChannelID, email, token, unsub, now)
			if err := uc.deps.Repo.Create(ctx, &s); err != nil {
				return err
			}
			send = &s
		} else if existing.Resubscribe(token, now) {
			existing.SalesChannelID = in.SalesChannelID
			if err := uc.deps.Repo.Update(ctx, existing); err != nil {
				return err
			}
			send = existing
		}
		if send != nil {
			storeName, err = uc.deps.Stores.StoreName(ctx, in.SalesChannelID)
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	if send == nil {
		return nil
	}
	if err := uc.deps.Mailer.Send(ctx, confirmationEmail(storeName, email,
		uc.deps.ConfirmURL(in.SalesChannelID, token), uc.deps.UnsubscribeURL(in.SalesChannelID, send.UnsubscribeToken))); err != nil {
		return fmt.Errorf("subscribe: send confirmation: %w", err)
	}
	return nil
}

func confirmationEmail(storeName, to, confirmURL, unsubscribeURL string) messaging.Message {
	name := html.EscapeString(storeName)
	return messaging.Message{
		Channel: messaging.ChannelEmail,
		To:      to,
		Subject: fmt.Sprintf("Confirm your subscription to %s", storeName),
		HTMLBody: fmt.Sprintf(`<p>Please confirm you'd like to receive emails from %s.</p>`+
			`<p><a href="%s">Yes, subscribe me</a></p>`+
			`<p>If you didn't sign up, ignore this email -- you won't be subscribed. The link works for 7 days.</p>`+
			`<p style="font-size:12px;color:#666"><a href="%s">Unsubscribe</a></p>`,
			name, html.EscapeString(confirmURL), html.EscapeString(unsubscribeURL)),
	}
}
