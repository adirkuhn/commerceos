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
// of use -- only needed to find the tenant's WhatsApp connection, same
// narrowing application/whatsapp's own use cases already do.
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
	db               *gorm.DB
	ruleRepo         mcDomain.AutomationRuleRepository
	templateRepo     mcDomain.MessageTemplateRepository
	salesOrderRepo   salesOrderFinder
	customerRepo     customerFinder
	connRepo         connectionLister
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory
	encryptionSecret string
	emailClient      messaging.Client
}

func NewEvaluateAutomationRulesUseCase(
	db *gorm.DB, ruleRepo mcDomain.AutomationRuleRepository, templateRepo mcDomain.MessageTemplateRepository,
	salesOrderRepo salesOrderFinder, customerRepo customerFinder, connRepo connectionLister,
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory, encryptionSecret string, emailClient messaging.Client,
) *EvaluateAutomationRulesUseCase {
	return &EvaluateAutomationRulesUseCase{
		db: db, ruleRepo: ruleRepo, templateRepo: templateRepo,
		salesOrderRepo: salesOrderRepo, customerRepo: customerRepo, connRepo: connRepo,
		evolutionFactory: evolutionFactory, encryptionSecret: encryptionSecret, emailClient: emailClient,
	}
}

func (uc *EvaluateAutomationRulesUseCase) Execute(ctx context.Context, subject, eventID string, payload map[string]any) error {
	var sends []pendingSend
	var whatsappConn *integrationsDomain.Connection

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
				to, err := uc.resolveRecipient(ctx, ch, customer, &whatsappConn)
				if err != nil {
					return err
				}
				if to == "" {
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

	return uc.dispatch(ctx, sends, whatsappConn)
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

// resolveRecipient returns the customer's destination address for ch, or ""
// if they have none on file (nothing to send to) -- the WhatsApp connection
// is looked up at most once per Execute call (cached into *whatsappConn),
// since a tenant has at most one.
func (uc *EvaluateAutomationRulesUseCase) resolveRecipient(ctx context.Context, ch messaging.Channel, customer *customersDomain.Customer, whatsappConn **integrationsDomain.Connection) (string, error) {
	switch ch {
	case messaging.ChannelEmail:
		return customer.Email, nil
	case messaging.ChannelWhatsApp:
		if customer.PhoneE164 == "" {
			return "", nil
		}
		if *whatsappConn == nil {
			conns, err := uc.connRepo.ListByTenant(ctx)
			if err != nil {
				return "", err
			}
			*whatsappConn = findWhatsAppConnection(conns)
		}
		if *whatsappConn == nil {
			return "", nil
		}
		return customer.PhoneE164, nil
	default:
		// SMS has no provider configured yet -- see
		// domain/messagecenter.KnownTriggerSubjects' own doc comment on
		// this project's staged scope.
		return "", nil
	}
}

// dispatch actually sends every gathered pendingSend, outside the DB
// transaction Execute opened above. Returns on the first failure (same
// "safe to retry, an occasional duplicate is a smaller harm than a silent
// drop" tradeoff every other consumer in this codebase already makes) --
// the whole event is nacked and redelivered, not partially retried.
func (uc *EvaluateAutomationRulesUseCase) dispatch(ctx context.Context, sends []pendingSend, whatsappConn *integrationsDomain.Connection) error {
	var whatsappClient integrationsDomain.EvolutionAPIClient
	for _, s := range sends {
		switch s.channel {
		case messaging.ChannelEmail:
			if err := uc.emailClient.Send(ctx, messaging.Message{Channel: messaging.ChannelEmail, To: s.to, Subject: s.subject, Body: s.body}); err != nil {
				return fmt.Errorf("evaluateAutomationRules: send email: %w", err)
			}
		case messaging.ChannelWhatsApp:
			if whatsappConn == nil {
				continue
			}
			if whatsappClient == nil {
				c, err := buildEvolutionClient(uc.evolutionFactory, uc.encryptionSecret, *whatsappConn)
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
