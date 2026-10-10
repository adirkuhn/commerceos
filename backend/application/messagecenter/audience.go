package messagecenter

import (
	"context"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
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

// audience is who an event's messages go to, plus the template values that
// come with them (e.g. customer_name).
type audience struct {
	recipients []recipient
	vars       map[string]any
}

// audienceResolver resolves one mcDomain.AudienceKind for an event.
type audienceResolver interface {
	resolve(ctx context.Context, a mcDomain.Audience, payload map[string]any) (audience, error)
}

// customerAudience is the customer of the event's sales order, found from
// a "sales_order_id" payload field. No order or no customer means nobody
// (rules still send to their extra recipients).
type customerAudience struct {
	salesOrderRepo salesOrderFinder
	customerRepo   customerFinder
}

func (r customerAudience) resolve(ctx context.Context, _ mcDomain.Audience, payload map[string]any) (audience, error) {
	customer, err := r.findCustomer(ctx, payload)
	if err != nil || customer == nil {
		return audience{vars: map[string]any{"customer_name": ""}}, err
	}
	return audience{
		recipients: []recipient{{email: customer.Email, phone: customer.PhoneE164}},
		vars:       map[string]any{"customer_name": customer.FirstName},
	}, nil
}

func (r customerAudience) findCustomer(ctx context.Context, payload map[string]any) (*customersDomain.Customer, error) {
	salesOrderID, ok := payload["sales_order_id"].(string)
	if !ok || salesOrderID == "" {
		return nil, nil
	}
	order, err := r.salesOrderRepo.FindByID(ctx, salesOrderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID == nil {
		return nil, nil
	}
	return r.customerRepo.FindByID(ctx, *order.CustomerID)
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

func (r staffAudience) resolve(ctx context.Context, a mcDomain.Audience, _ map[string]any) (audience, error) {
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
