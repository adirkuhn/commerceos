package messagecenter

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/tenantemail"
	"log"
	"strings"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// salesOrderFinder is oms.SalesOrderRepository narrowed to point of use
// (ISP) -- EvaluateAutomationRulesUseCase only ever needs to resolve one
// order's customer, given the triggering event's sales_order_id.
type salesOrderFinder interface {
	FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error)
}

// customerFinder is customers.CustomerRepository narrowed to point of use.
type customerFinder interface {
	FindByID(ctx context.Context, id string) (*customersDomain.Customer, error)
}

// connectionLister is integrations.ConnectionRepository narrowed to point
// of use -- finds the tenant's WhatsApp/Email connections, same narrowing
// application/whatsapp's own use cases already do.
type connectionLister interface {
	ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error)
}

// pendingSend is one resolved (channel, recipient, rendered content) tuple,
// gathered while the DB transaction is open and actually sent afterwards --
// same "read inside the transaction, send after it closes" split as
// NotifyShipmentDeliveryUseCase, so a slow network call to an email/WhatsApp
// provider never holds a DB transaction open.
type pendingSend struct {
	channel messaging.Channel
	to      string
	subject string
	body    string
}

// EvaluateAutomationRulesUseCase is what every automation-engine consumer
// (infrastructure/consumers.AutomationEngineConsumer, one per trigger
// subject) calls on each received event: find every enabled AutomationRule
// for this subject, keep the ones whose Conditions match the event's own
// payload, and send each matched rule's MessageTemplate.
//
// Who it goes to is the trigger's audience (mcDomain.AudienceFor: the
// order's customer, or for staff triggers the notification recipients of a
// category), resolved by one audienceResolver per audience kind, plus the
// rule's ExtraRecipients on the email channel. A trigger whose audience
// resolves to nobody still sends to those extra recipients.
type EvaluateAutomationRulesUseCase struct {
	db                 *gorm.DB
	ruleRepo           mcDomain.AutomationRuleRepository
	templateRepo       mcDomain.MessageTemplateRepository
	audiences          map[mcDomain.AudienceKind]audienceResolver
	connRepo           connectionLister
	evolutionFactory   integrationsDomain.EvolutionAPIClientFactory
	encryptionSecret   string
	emailClientFactory messaging.EmailClientFactory
	// fallbackEmailClient is used when the tenant has no connected
	// domain/integrations.ProviderEmail integration -- the platform-level
	// SMTP client (or noop) cmd/scheduler/main.go already builds from
	// SMTP_HOST/etc env vars, same fallback every automation email used
	// before per-tenant email connections existed.
	fallbackEmailClient messaging.Client
}

func NewEvaluateAutomationRulesUseCase(
	db *gorm.DB, ruleRepo mcDomain.AutomationRuleRepository, templateRepo mcDomain.MessageTemplateRepository,
	salesOrderRepo salesOrderFinder, customerRepo customerFinder, connRepo connectionLister,
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory, encryptionSecret string,
	emailClientFactory messaging.EmailClientFactory, fallbackEmailClient messaging.Client,
	staffRecipients categoryEmailRecipients,
) *EvaluateAutomationRulesUseCase {
	return &EvaluateAutomationRulesUseCase{
		db: db, ruleRepo: ruleRepo, templateRepo: templateRepo,
		audiences: map[mcDomain.AudienceKind]audienceResolver{
			mcDomain.AudienceCustomer: customerAudience{salesOrderRepo: salesOrderRepo, customerRepo: customerRepo},
			mcDomain.AudienceStaff:    staffAudience{recipients: staffRecipients},
		},
		connRepo:         connRepo,
		evolutionFactory: evolutionFactory, encryptionSecret: encryptionSecret,
		emailClientFactory: emailClientFactory, fallbackEmailClient: fallbackEmailClient,
	}
}

func (uc *EvaluateAutomationRulesUseCase) Execute(ctx context.Context, subject, eventID string, payload map[string]any) error {
	var sends []pendingSend
	var conns []integrationsDomain.Connection

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		rules, err := uc.ruleRepo.ListActiveBySubject(ctx, subject)
		if err != nil {
			return err
		}
		if len(rules) == 0 {
			return nil
		}

		target := mcDomain.AudienceFor(subject)
		resolver, ok := uc.audiences[target.Kind]
		if !ok {
			return fmt.Errorf("no audience resolver for %q", target.Kind)
		}
		aud, err := resolver.resolve(ctx, target, payload)
		if err != nil {
			return err
		}
		data := templateData(payload, aud.vars)

		conns, err = uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		whatsappConnected := findWhatsAppConnection(conns) != nil

		for _, rule := range rules {
			if !rule.Matches(payload) {
				continue
			}
			template, err := uc.templateRepo.FindByID(ctx, rule.TemplateID)
			if err != nil {
				return err
			}
			if !template.Enabled {
				continue
			}
			channels := template.Channels
			if rule.ChannelOverride != nil {
				channels = []messaging.Channel{*rule.ChannelOverride}
			}
			for _, ch := range channels {
				if ch == messaging.ChannelWhatsApp && !whatsappConnected {
					continue
				}
				addresses := channelAddresses(ch, aud.recipients, rule.ExtraRecipients)
				if len(addresses) == 0 {
					continue
				}
				subjectLine, body := renderSubject(template.Subject, data), renderBody(template.Body, data, ch)
				for _, to := range addresses {
					sends = append(sends, pendingSend{channel: ch, to: to, subject: subjectLine, body: body})
				}
			}
		}
		if len(sends) == 0 {
			log.Printf("evaluateAutomationRules: subject %s event %s matched rules but resolved no recipient, skipping send", subject, eventID)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("evaluateAutomationRules: %w", err)
	}

	return uc.dispatch(ctx, sends, conns)
}

// channelAddresses is every distinct address one channel's message goes
// to: the audience's, then -- email only -- the rule's extra recipients.
// Duplicates (case-insensitive) are sent once.
func channelAddresses(ch messaging.Channel, recipients []recipient, extras []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(addr string) {
		key := strings.ToLower(addr)
		if addr == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, addr)
	}
	for _, r := range recipients {
		add(r.address(ch))
	}
	if ch == messaging.ChannelEmail {
		for _, e := range extras {
			add(e)
		}
	}
	return out
}

// dispatch actually sends every gathered pendingSend, outside the DB
// transaction Execute opened above. Returns on the first failure (same
// "safe to retry, an occasional duplicate is a smaller harm than a silent
// drop" tradeoff every other consumer in this codebase already makes) --
// the whole event is nacked and redelivered, not partially retried.
func (uc *EvaluateAutomationRulesUseCase) dispatch(ctx context.Context, sends []pendingSend, conns []integrationsDomain.Connection) error {
	var whatsappClient integrationsDomain.EvolutionAPIClient
	var emailClient messaging.Client

	for _, s := range sends {
		switch s.channel {
		case messaging.ChannelEmail:
			if emailClient == nil {
				c, err := uc.resolveEmailClient(conns)
				if err != nil {
					return fmt.Errorf("evaluateAutomationRules: %w", err)
				}
				emailClient = c
			}
			// smtp.Client.Send only reads HTMLBody, never Body (see
			// infrastructure/messaging/smtp.Client.Send) -- Body alone
			// here would silently send an empty message.
			if err := emailClient.Send(ctx, messaging.Message{Channel: messaging.ChannelEmail, To: s.to, Subject: s.subject, HTMLBody: s.body}); err != nil {
				return fmt.Errorf("evaluateAutomationRules: send email: %w", err)
			}
		case messaging.ChannelWhatsApp:
			if whatsappClient == nil {
				conn := findWhatsAppConnection(conns)
				if conn == nil {
					continue
				}
				c, err := buildEvolutionClient(uc.evolutionFactory, uc.encryptionSecret, *conn)
				if err != nil {
					return fmt.Errorf("evaluateAutomationRules: %w", err)
				}
				whatsappClient = c
			}
			if err := whatsappClient.SendText(ctx, s.to, s.body); err != nil {
				return fmt.Errorf("evaluateAutomationRules: send whatsapp: %w", err)
			}
		}
	}
	return nil
}

// resolveEmailClient prefers the tenant's own connected email integration
// (domain/integrations.ProviderEmail -- see that constant's doc comment for
// the Connection field mapping) over fallbackEmailClient, the injected
// platform-level SMTP client every automation email used before per-tenant
// email connections existed.
func (uc *EvaluateAutomationRulesUseCase) resolveEmailClient(conns []integrationsDomain.Connection) (messaging.Client, error) {
	return tenantemail.Resolve(conns, uc.encryptionSecret, uc.emailClientFactory, uc.fallbackEmailClient)
}
