// Package whatsapp is the WhatsApp sales-channel integration's application
// layer: inbound message dispatch (this file) and the product-info lookup
// it delegates to. Talks to the gateway (Evolution API, an unofficial,
// self-hosted WhatsApp Web bridge -- see
// domain/integrations.ProviderWhatsApp) only through the
// EvolutionAPIClientFactory port, never a concrete client.
package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// whatsappAIIntentTimeout bounds the AI-assisted classification call --
// a customer waiting on a WhatsApp reply must never be stuck behind a
// slow/hanging third-party AI provider; on timeout the message falls back
// to the normal help text, same as any other AI failure (see
// tryAIAssistedReply).
const whatsappAIIntentTimeout = 8 * time.Second

// productInfoLookup is *LookupProductInfoUseCase narrowed to point of use
// (ISP) -- lets HandleInboundMessageUseCase be unit tested without a real
// DB or product/inventory repos.
type productInfoLookup interface {
	Execute(ctx context.Context, skuCode string) (*ProductInfo, error)
}

// orderFlowHandler is *OrderFlowUseCase narrowed to point of use (ISP).
// handled=false means text wasn't part of an order flow (an Idle
// conversation given a command OrderFlowUseCase doesn't own) -- buildReply
// then falls through to its own STOCK/PRICE/EXPIRY/INFO/HELP handling.
type orderFlowHandler interface {
	Execute(ctx context.Context, phone string, customer *customersDomain.Customer, text string) (reply string, handled bool, err error)
}

// messageRecorder is whatsappDomain.MessageRepository narrowed to point of
// use (ISP) -- HandleInboundMessageUseCase only ever writes.
type messageRecorder interface {
	Create(ctx context.Context, m *whatsappDomain.Message) error
}

// aiIntentResolver is whatsappDomain.AIIntentParserResolver narrowed to
// point of use (ISP).
type aiIntentResolver interface {
	Resolve(ctx context.Context) (whatsappDomain.AIIntentParser, error)
}

// skuQueryResolver is *ResolveSKUFromQueryUseCase narrowed to point of use
// (ISP).
type skuQueryResolver interface {
	Execute(ctx context.Context, query string) (string, error)
}

// HandleInboundMessageUseCase is the WhatsApp branch of the webhook cold
// path's per-topic dispatch (see application/webhooks.ProcessWebhookEventUseCase,
// registered under provider key "whatsapp"). A message that doesn't parse
// into a known command still gets a helpful reply -- Execute only returns
// an error for infra failures (DB, decrypt, or the outbound send itself),
// never for "the customer typed something we don't understand", since that
// must not enter the webhook consumer's nack/retry loop.
type HandleInboundMessageUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory
	encryptionSecret string
	lookupProduct    productInfoLookup
	customerRepo     customersDomain.CustomerRepository
	orderFlow        orderFlowHandler
	messageRepo      messageRecorder
	aiIntentResolver aiIntentResolver
	resolveSKU       skuQueryResolver
}

func NewHandleInboundMessageUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory,
	encryptionSecret string,
	lookupProduct productInfoLookup,
	customerRepo customersDomain.CustomerRepository,
	orderFlow orderFlowHandler,
	messageRepo messageRecorder,
	aiIntentResolver aiIntentResolver,
	resolveSKU skuQueryResolver,
) *HandleInboundMessageUseCase {
	return &HandleInboundMessageUseCase{
		db: db, connRepo: connRepo, evolutionFactory: evolutionFactory, encryptionSecret: encryptionSecret,
		lookupProduct: lookupProduct, customerRepo: customerRepo, orderFlow: orderFlow, messageRepo: messageRepo,
		aiIntentResolver: aiIntentResolver, resolveSKU: resolveSKU,
	}
}

func customerIDPtr(customer *customersDomain.Customer) *string {
	if customer == nil {
		return nil
	}
	id := customer.ID
	return &id
}

func (uc *HandleInboundMessageUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	var payload evolutionAPIMessage
	if err := json.Unmarshal(envelope.RawPayload, &payload); err != nil {
		return fmt.Errorf("handleInboundMessage: decode payload: %w", err)
	}
	if payload.Data.Info.IsFromMe {
		return nil
	}
	if payload.isGroup() {
		return nil
	}
	phone := payload.phone()
	if phone == "" {
		return nil
	}

	ctx = tenant.WithTenantID(ctx, envelope.TenantID)

	var conn *integrationsDomain.Connection
	var customer *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conns, err := uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		conn = findWhatsAppConnection(conns)
		if conn != nil {
			customer, err = uc.customerRepo.FindByPhoneE164(ctx, phone)
			if err != nil {
				return err
			}
		}
		return uc.messageRepo.Create(ctx, &whatsappDomain.Message{
			TenantID: envelope.TenantID, PhoneE164: phone, CustomerID: customerIDPtr(customer),
			Direction: whatsappDomain.MessageDirectionInbound, Body: payload.text(),
			WebhookEventID: &envelope.EventID,
		})
	})
	if err != nil {
		return fmt.Errorf("handleInboundMessage: load connection/customer: %w", err)
	}
	if conn == nil {
		// A whatsapp Connection must exist for this webhook to have been
		// verified at all (see application/webhooks.ReceiveWebhookUseCase's
		// tenantWebhookSecret) -- reaching here with none found means the
		// connection was deleted between verification and dispatch. Nothing
		// to reply through; drop it.
		return nil
	}

	client, err := buildEvolutionClient(uc.evolutionFactory, uc.encryptionSecret, *conn)
	if err != nil {
		return fmt.Errorf("handleInboundMessage: %w", err)
	}

	reply, err := uc.buildReply(ctx, conn, customer, phone, payload.text())
	if err != nil {
		return fmt.Errorf("handleInboundMessage: build reply: %w", err)
	}

	if err := client.SendText(ctx, phone, reply); err != nil {
		return fmt.Errorf("handleInboundMessage: send reply: %w", err)
	}

	// Logged best-effort, after the send: the reply is already on its way
	// to the customer, so a logging failure here must not turn into a
	// webhook nack -- that would redeliver this event and send a second,
	// duplicate reply.
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.messageRepo.Create(ctx, &whatsappDomain.Message{
			TenantID: envelope.TenantID, PhoneE164: phone, CustomerID: customerIDPtr(customer),
			Direction: whatsappDomain.MessageDirectionOutbound, Body: reply,
		})
	})
	if err != nil {
		log.Printf("handleInboundMessage: log outbound message: %v", err)
	}
	return nil
}

const helpText = "Commands:\nSTOCK <code> - check stock\nPRICE <code> - check price\nEXPIRY <code> - check nearest expiry\nINFO <code> - stock, price and expiry\nORDER <code> <quantity> - add to cart\nCART - view your cart\nCHECKOUT - place your order\nCANCEL - clear your cart"

// buildReply's only error return is an infra failure from the product
// lookup (DB down, etc) -- an expected "business" outcome (unknown
// command, product not found/not enrolled, no whatsapp channel configured)
// always produces a reply string instead, never an error, since those must
// reach the customer as a normal message, not trigger the webhook
// consumer's nack/retry loop.
func (uc *HandleInboundMessageUseCase) buildReply(ctx context.Context, conn *integrationsDomain.Connection, customer *customersDomain.Customer, phone, text string) (string, error) {
	if uc.orderFlow != nil {
		reply, handled, err := uc.orderFlow.Execute(ctx, phone, customer, text)
		if err != nil {
			return "", err
		}
		if handled {
			return greet(customer, reply), nil
		}
	}

	if command, skuCode, ok := parseRigidLookupCommand(text); ok {
		return uc.replyForLookupCommand(ctx, customer, command, skuCode)
	}

	// The message didn't match "COMMAND CODE" -- try AI-assisted
	// classification (STOCK/PRICE/EXPIRY/INFO only, see
	// tryAIAssistedReply) when this tenant has opted in. Any AI-side
	// failure (resolver, network, timeout, malformed response) degrades
	// silently to the same help text a rigid-parse miss already produces
	// -- an assist feature must never turn into an error, and must never
	// risk a duplicate reply from a webhook retry.
	if conn.WhatsAppAIParsingEnabled {
		if reply, ok, err := uc.tryAIAssistedReply(ctx, customer, text); err != nil {
			return "", err
		} else if ok {
			return reply, nil
		}
	}

	return greet(customer, helpText), nil
}

// parseRigidLookupCommand is the original, deterministic "COMMAND CODE"
// parser -- STOCK/PRICE/EXPIRY/INFO only (ORDER/CART/CHECKOUT/CANCEL are
// handled earlier, by orderFlowHandler). ok=false for anything else,
// including a recognized word with no second field.
func parseRigidLookupCommand(text string) (command, skuCode string, ok bool) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return "", "", false
	}
	switch cmd := strings.ToUpper(fields[0]); cmd {
	case "STOCK", "PRICE", "EXPIRY", "INFO":
		return cmd, fields[1], true
	default:
		return "", "", false
	}
}

// replyForLookupCommand is the shared "look up this SKU code and format
// the answer" step both the rigid parser and the AI-assisted path funnel
// into -- the AI never composes reply text itself, it only ever supplies
// (command, skuCode) inputs to this exact same deterministic formatter.
func (uc *HandleInboundMessageUseCase) replyForLookupCommand(ctx context.Context, customer *customersDomain.Customer, command, skuCode string) (string, error) {
	info, err := uc.lookupProduct.Execute(ctx, skuCode)
	var reply string
	switch {
	case err == nil:
		reply = formatProductInfo(command, *info)
	case errors.Is(err, ErrProductNotFound):
		reply = fmt.Sprintf("We couldn't find a product with code %s.", skuCode)
	case errors.Is(err, ErrNoWhatsAppChannel):
		reply = "WhatsApp ordering isn't set up for this account yet."
	default:
		return "", err
	}
	return greet(customer, reply), nil
}

// tryAIAssistedReply classifies text with the tenant's connected AI
// provider, resolves the extracted product query to a concrete SKU code
// (ResolveSKUFromQueryUseCase -- exact code first, then a strict fuzzy
// name match), and formats the reply through the exact same
// replyForLookupCommand every rigid command uses. ok=false means "AI
// assist had nothing usable" (unconfigured, Unknown intent, or any
// failure short of a real product-lookup infra error) -- the caller falls
// back to help text in that case, never to an error.
func (uc *HandleInboundMessageUseCase) tryAIAssistedReply(ctx context.Context, customer *customersDomain.Customer, text string) (reply string, ok bool, err error) {
	if uc.aiIntentResolver == nil || uc.resolveSKU == nil {
		return "", false, nil
	}
	parser, err := uc.aiIntentResolver.Resolve(ctx)
	if err != nil {
		log.Printf("handleInboundMessage: resolve AI intent parser: %v", err)
		return "", false, nil
	}
	if parser == nil {
		// Toggle is on but no AI connection exists -- shouldn't happen
		// (SetWhatsAppAIParsingUseCase guards it), but never treat it as
		// a dispatch error.
		return "", false, nil
	}

	aiCtx, cancel := context.WithTimeout(ctx, whatsappAIIntentTimeout)
	defer cancel()
	intent, err := parser.ParseIntent(aiCtx, text)
	if err != nil {
		log.Printf("handleInboundMessage: AI intent parse failed: %v", err)
		return "", false, nil
	}
	if intent.Command == whatsappDomain.IntentCommandUnknown || intent.ProductQuery == "" {
		return "", false, nil
	}

	skuCode, err := uc.resolveSKU.Execute(ctx, intent.ProductQuery)
	if err != nil {
		return "", false, err
	}
	if skuCode == "" {
		reply := fmt.Sprintf("We couldn't find a product matching %q.", intent.ProductQuery)
		return greet(customer, reply), true, nil
	}

	reply, err = uc.replyForLookupCommand(ctx, customer, string(intent.Command), skuCode)
	if err != nil {
		return "", false, err
	}
	return reply, true, nil
}

// greet prefixes reply with the customer's name -- only for a known
// customer (matched by PhoneE164, see HandleInboundMessageUseCase.Execute);
// an unrecognized number gets the same reply with no personalization,
// never a hint about whether that number is a customer or not.
func greet(customer *customersDomain.Customer, reply string) string {
	if customer != nil && customer.FirstName != "" {
		return fmt.Sprintf("Hi %s! %s", customer.FirstName, reply)
	}
	return reply
}

func formatProductInfo(command string, info ProductInfo) string {
	var lines []string
	if command == "STOCK" || command == "INFO" {
		lines = append(lines, fmt.Sprintf("%s (%s) - in stock: %d", info.ProductName, info.SKUCode, info.StockQuantity))
	}
	if command == "PRICE" || command == "INFO" {
		price := fmt.Sprintf("%.2f %s", float64(info.PriceMinorUnits)/100, info.CurrencyCode)
		if info.SalePriceMinorUnits != nil {
			price = fmt.Sprintf("%.2f %s (was %.2f)", float64(*info.SalePriceMinorUnits)/100, info.CurrencyCode, float64(info.PriceMinorUnits)/100)
		}
		lines = append(lines, fmt.Sprintf("Price: %s", price))
	}
	if command == "EXPIRY" || command == "INFO" {
		if info.NearestExpiry != nil {
			lines = append(lines, fmt.Sprintf("Nearest expiry: %s", info.NearestExpiry.Format("2006-01-02")))
		} else {
			lines = append(lines, "No expiry-tracked batch for this product.")
		}
	}
	return strings.Join(lines, "\n")
}
