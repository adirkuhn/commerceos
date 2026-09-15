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
	"strings"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// evolutionAPIMessage is Evolution API's "messages.upsert" webhook payload --
// built from its published docs, NOT confirmed against a real running
// instance, same caveat as infrastructure/evolutionapi's package doc
// comment. RemoteJID is "<phone>@s.whatsapp.net"; FromMe is true for a
// message this same WhatsApp number sent (including our own outbound
// replies echoed back by some gateway configurations) and must never be
// treated as an inbound command, or a reply could recursively trigger
// another reply.
type evolutionAPIMessage struct {
	Data struct {
		Key struct {
			RemoteJID string `json:"remoteJid"`
			FromMe    bool   `json:"fromMe"`
		} `json:"key"`
		Message struct {
			Conversation        string `json:"conversation"`
			ExtendedTextMessage struct {
				Text string `json:"text"`
			} `json:"extendedTextMessage"`
		} `json:"message"`
	} `json:"data"`
}

func (m evolutionAPIMessage) phone() string {
	return strings.SplitN(m.Data.Key.RemoteJID, "@", 2)[0]
}

// isGroup is true for a group chat's RemoteJID ("<groupId>@g.us"), as
// opposed to a direct 1-1 chat ("<phone>@s.whatsapp.net"). Group messages
// are deliberately unsupported (no bot commands, no order flow, no
// customer lookup by "phone" -- a group JID's numeric prefix is a group
// id, not anyone's phone number) and must be dropped before Execute does
// anything with phone().
func (m evolutionAPIMessage) isGroup() bool {
	return strings.HasSuffix(m.Data.Key.RemoteJID, "@g.us")
}

func (m evolutionAPIMessage) text() string {
	if m.Data.Message.Conversation != "" {
		return m.Data.Message.Conversation
	}
	return m.Data.Message.ExtendedTextMessage.Text
}

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
}

func NewHandleInboundMessageUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	evolutionFactory integrationsDomain.EvolutionAPIClientFactory,
	encryptionSecret string,
	lookupProduct productInfoLookup,
	customerRepo customersDomain.CustomerRepository,
	orderFlow orderFlowHandler,
) *HandleInboundMessageUseCase {
	return &HandleInboundMessageUseCase{
		db: db, connRepo: connRepo, evolutionFactory: evolutionFactory, encryptionSecret: encryptionSecret,
		lookupProduct: lookupProduct, customerRepo: customerRepo, orderFlow: orderFlow,
	}
}

func (uc *HandleInboundMessageUseCase) Execute(ctx context.Context, envelope webhooks.Envelope) error {
	var payload evolutionAPIMessage
	if err := json.Unmarshal(envelope.RawPayload, &payload); err != nil {
		return fmt.Errorf("handleInboundMessage: decode payload: %w", err)
	}
	if payload.Data.Key.FromMe {
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
		if conn == nil {
			return nil
		}
		customer, err = uc.customerRepo.FindByPhoneE164(ctx, phone)
		return err
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

	reply, err := uc.buildReply(ctx, customer, phone, payload.text())
	if err != nil {
		return fmt.Errorf("handleInboundMessage: build reply: %w", err)
	}

	if err := client.SendText(ctx, phone, reply); err != nil {
		return fmt.Errorf("handleInboundMessage: send reply: %w", err)
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
func (uc *HandleInboundMessageUseCase) buildReply(ctx context.Context, customer *customersDomain.Customer, phone, text string) (string, error) {
	if uc.orderFlow != nil {
		reply, handled, err := uc.orderFlow.Execute(ctx, phone, customer, text)
		if err != nil {
			return "", err
		}
		if handled {
			return greet(customer, reply), nil
		}
	}

	fields := strings.Fields(text)
	if len(fields) < 2 {
		return greet(customer, helpText), nil
	}
	command := strings.ToUpper(fields[0])
	skuCode := fields[1]

	var reply string
	switch command {
	case "STOCK", "PRICE", "EXPIRY", "INFO":
		info, err := uc.lookupProduct.Execute(ctx, skuCode)
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
	default:
		reply = helpText
	}
	return greet(customer, reply), nil
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
