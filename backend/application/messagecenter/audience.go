package messagecenter

import (
	"context"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

// recipient is one person a rule's message can go to; an empty address
// means none on file for that channel.
type recipient struct {
	email string
	phone string
}

func (r recipient) address(ch messaging.Channel) string {
	switch ch {
	case messaging.ChannelEmail:
		return r.email
	case messaging.ChannelWhatsApp:
		return r.phone
	default:
		// SMS has no provider configured yet.
		return ""
	}
}

// audience is who an event's messages go to.
type audience struct {
	recipients []recipient
}

// audienceResolver resolves one mcDomain.AudienceKind for an event; order
// is the trigger's sales order (nil when it has none).
type audienceResolver interface {
	resolve(ctx context.Context, a mcDomain.Audience, order *loadedOrder) (audience, error)
}

// customerAudience is the customer of the trigger's sales order -- the
// order's billing contact for a guest order. No order means nobody (rules
// still send to their extra recipients).
type customerAudience struct{}

func (customerAudience) resolve(_ context.Context, _ mcDomain.Audience, order *loadedOrder) (audience, error) {
	if order == nil {
		return audience{}, nil
	}
	r := recipient{email: order.order.Billing.Email}
	if c := order.customer; c != nil {
		r.phone = c.PhoneE164
		if c.Email != "" {
			r.email = c.Email
		}
	}
	return audience{recipients: []recipient{r}}, nil
}

// categoryEmailRecipients is application/notifications'
// CategoryEmailRecipientsUseCase narrowed to point of use.
type categoryEmailRecipients interface {
	Execute(ctx context.Context, c notificationsDomain.Category) ([]string, error)
}

// staffAudience is the tenant's team: the notification email recipients of
// the audience's category, or nobody when that category's email is off.
type staffAudience struct {
	recipients categoryEmailRecipients
}

func (r staffAudience) resolve(ctx context.Context, a mcDomain.Audience, _ *loadedOrder) (audience, error) {
	emails, err := r.recipients.Execute(ctx, notificationsDomain.Category(a.StaffCategory))
	if err != nil {
		return audience{}, err
	}
	out := make([]recipient, len(emails))
	for i, e := range emails {
		out[i] = recipient{email: e}
	}
	return audience{recipients: out}, nil
}
