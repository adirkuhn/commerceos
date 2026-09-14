package messagecenter

import (
	"context"
	"fmt"
	"log"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/crypto"
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
// Recipient resolution is deliberately generic, not per-subject: any event
// payload carrying a "sales_order_id" field (order/shipment/refund events)
// resolves to that order's Customer; every other trigger subject (product/
// category/integration/inventory events -- merchant/ops events with no
// customer to notify) matches its rules but sends nothing, logged clearly
// rather than guessing at a recipient. A future trigger subject with its
// own payload shape carrying sales_order_id gets recipient resolution for
// free, no new branch needed here.
type EvaluateAutomationRulesUseCase struct {
	db                 *gorm.DB
	ruleRepo           mcDomain.AutomationRuleRepository
	templateRepo       mcDomain.MessageTemplateRepository
	salesOrderRepo     salesOrderFinder
	customerRepo       customerFinder
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
) *EvaluateAutomationRulesUseCase {
	return &EvaluateAutomationRulesUseCase{
		db: db, ruleRepo: ruleRepo, templateRepo: templateRepo,
		salesOrderRepo: salesOrderRepo, customerRepo: customerRepo, connRepo: connRepo,
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

		customer, err := uc.resolveCustomer(ctx, payload)
		if err != nil {
			return err
		}
		if customer == nil {
			log.Printf("evaluateAutomationRules: subject %s event %s has no resolvable customer recipient, skipping send", subject, eventID)
			return nil
		}
		vars := buildTemplateVars(payload, customer.FirstName)

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
				to := recipientAddress(ch, customer)
				if to == "" {
					continue
				}
				if ch == messaging.ChannelWhatsApp && !whatsappConnected {
					continue
				}
				sends = append(sends, pendingSend{
					channel: ch, to: to,
					subject: RenderTemplate(template.Subject, vars),
					body:    RenderTemplate(template.Body, vars),
				})
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("evaluateAutomationRules: %w", err)
	}

	return uc.dispatch(ctx, sends, conns)
}

// resolveCustomer resolves the triggering event's customer, generically,
// from a "sales_order_id" payload field -- see this use case's own doc
// comment for why this is the one recipient-resolution rule, not one per
// trigger subject.
func (uc *EvaluateAutomationRulesUseCase) resolveCustomer(ctx context.Context, payload map[string]any) (*customersDomain.Customer, error) {
	salesOrderID, ok := payload["sales_order_id"].(string)
	if !ok || salesOrderID == "" {
		return nil, nil
	}
	order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
	if err != nil {
		return nil, err
	}
	if order.CustomerID == nil {
		return nil, nil
	}
	return uc.customerRepo.FindByID(ctx, *order.CustomerID)
}

// recipientAddress returns the customer's destination address for ch, or ""
// if they have none on file (nothing to send to). Pure -- no I/O -- since
// which connection sends it, if any, is dispatch's concern, not this one's.
func recipientAddress(ch messaging.Channel, customer *customersDomain.Customer) string {
	switch ch {
	case messaging.ChannelEmail:
		return customer.Email
	case messaging.ChannelWhatsApp:
		return customer.PhoneE164
	default:
		// SMS has no provider configured yet -- see
		// domain/messagecenter.KnownTriggerSubjects' own doc comment on
		// this project's staged scope.
		return ""
	}
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
	for i := range conns {
		if conns[i].Provider != integrationsDomain.ProviderEmail {
			continue
		}
		conn := conns[i]
		password, err := crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt email connection secret: %w", err)
		}
		port := ""
		if conn.SMTPPort != nil {
			port = *conn.SMTPPort
		}
		from := ""
		if conn.EmailFrom != nil {
			from = *conn.EmailFrom
		}
		return uc.emailClientFactory.NewClient(conn.StoreURL.String(), port, conn.ConsumerKey, password, from), nil
	}
	return uc.fallbackEmailClient, nil
}
