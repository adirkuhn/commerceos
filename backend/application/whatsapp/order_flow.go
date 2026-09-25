package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ErrChannelWarehouseNotConfigured guards placeOrder: sales_orders.warehouse_id
// is a NOT NULL FK, so the tenant's WhatsApp SalesChannel must have a
// DefaultWarehouseID set before an order can be placed through it -- found
// live via the equivalent internal-storefront checkout bug, see
// application/storefront.ErrChannelWarehouseNotConfigured's doc comment.
var ErrChannelWarehouseNotConfigured = errors.New("whatsapp: no default warehouse configured for this channel")

// salesOrderCreator is *application/oms.CreateSalesOrderUseCase narrowed to
// point of use (ISP).
type salesOrderCreator interface {
	Execute(ctx context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error)
}

// customerCreator is *application/customers.CreateCustomerUseCase narrowed
// to point of use (ISP).
type customerCreator interface {
	Execute(ctx context.Context, in appcustomers.CreateCustomerInput) (*customersDomain.Customer, error)
}

// paymentLinkGenerator is *GeneratePaymentLinkUseCase narrowed to point of
// use (ISP) -- may be nil (a deployment that hasn't wired payment yet),
// same optional-dependency convention as application/shipping's
// ShipmentDeliveryNotifier.
type paymentLinkGenerator interface {
	Execute(ctx context.Context, amountMinorUnits int64, currencyCode, description, reference string) (*integrationsDomain.PaymentLink, bool, error)
}

// OrderFlowUseCase is HandleInboundMessageUseCase's order-placement branch:
// ORDER/CART/CHECKOUT and everything a Conversation's in-progress state
// implies about how to interpret the next message (a capture answer, or a
// YES/NO confirmation). Returns handled=false for any text that isn't part
// of an order flow (an Idle conversation given a command this use case
// doesn't own), letting the caller fall through to its own STOCK/PRICE/
// EXPIRY/INFO/HELP handling -- same "handled bool" composition
// HandleInboundMessageUseCase already uses nowhere else, introduced here
// because this is the first place two independent command sets need to
// share one inbound message.
type OrderFlowUseCase struct {
	db                  *gorm.DB
	conversationRepo    whatsappDomain.ConversationRepository
	channelRepo         channelsDomain.SalesChannelRepository
	lookupProduct       productInfoLookup
	createSalesOrder    salesOrderCreator
	createCustomer      customerCreator
	generatePaymentLink paymentLinkGenerator
}

func NewOrderFlowUseCase(
	db *gorm.DB,
	conversationRepo whatsappDomain.ConversationRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	lookupProduct productInfoLookup,
	createSalesOrder salesOrderCreator,
	createCustomer customerCreator,
	generatePaymentLink paymentLinkGenerator,
) *OrderFlowUseCase {
	return &OrderFlowUseCase{
		db: db, conversationRepo: conversationRepo, channelRepo: channelRepo, lookupProduct: lookupProduct,
		createSalesOrder: createSalesOrder, createCustomer: createCustomer,
		generatePaymentLink: generatePaymentLink,
	}
}

// findActiveWhatsAppChannel mirrors LookupProductInfoUseCase's own method of
// the same name -- deliberately a local copy rather than a shared helper,
// same "small local copies, not premature sharing" convention this
// package's other use cases already follow.
func (uc *OrderFlowUseCase) findActiveWhatsAppChannel(ctx context.Context) (*channelsDomain.SalesChannel, error) {
	channels, err := uc.channelRepo.ListByTenant(ctx)
	if err != nil {
		return nil, err
	}
	for i := range channels {
		if channels[i].Type == channelsDomain.ChannelTypeWhatsApp && channels[i].Active {
			return &channels[i], nil
		}
	}
	return nil, ErrNoWhatsAppChannel
}

// resolveWarehouse looks up the tenant's WhatsApp channel's configured
// SalesChannel.DefaultWarehouseID -- see ErrChannelWarehouseNotConfigured's
// doc comment for why this must be checked before placeOrder ever calls
// createSalesOrder.
func (uc *OrderFlowUseCase) resolveWarehouse(ctx context.Context) (string, error) {
	var warehouseID string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channel, err := uc.findActiveWhatsAppChannel(ctx)
		if err != nil {
			return err
		}
		if channel.DefaultWarehouseID == nil {
			return ErrChannelWarehouseNotConfigured
		}
		warehouseID = *channel.DefaultWarehouseID
		return nil
	})
	return warehouseID, err
}

func (uc *OrderFlowUseCase) Execute(ctx context.Context, phone string, customer *customersDomain.Customer, text string) (reply string, handled bool, err error) {
	conv, err := uc.loadConversation(ctx, phone)
	if err != nil {
		return "", false, fmt.Errorf("orderFlow: load conversation: %w", err)
	}

	fields := strings.Fields(text)
	command := ""
	if len(fields) > 0 {
		command = strings.ToUpper(fields[0])
	}

	// CANCEL interrupts any state -- must be checked before the
	// state-based routing below, or a customer trying to bail out of a
	// capture step would have their "CANCEL" swallowed as if it were their
	// name/email/address.
	if command == "CANCEL" {
		return uc.handleCancel(ctx, conv)
	}

	switch conv.State {
	case whatsappDomain.ConversationStateAwaitingNewCustomerName:
		return uc.handleCaptureName(ctx, conv, text)
	case whatsappDomain.ConversationStateAwaitingNewCustomerEmail:
		return uc.handleCaptureEmail(ctx, conv, text)
	case whatsappDomain.ConversationStateAwaitingNewCustomerAddress:
		return uc.handleCaptureAddress(ctx, conv, text)
	case whatsappDomain.ConversationStateAwaitingOrderConfirmation:
		return uc.handleConfirmation(ctx, conv, customer, phone, text)
	}

	switch command {
	case "ORDER":
		return uc.handleAddToCart(ctx, conv, fields)
	case "CART":
		return uc.handleShowCart(conv)
	case "CHECKOUT":
		return uc.handleCheckout(ctx, conv, customer)
	default:
		return "", false, nil
	}
}

func (uc *OrderFlowUseCase) loadConversation(ctx context.Context, phone string) (*whatsappDomain.Conversation, error) {
	var conv *whatsappDomain.Conversation
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.conversationRepo.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		conv = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	if conv == nil {
		conv = &whatsappDomain.Conversation{TenantID: tenant.IDFromContext(ctx), PhoneE164: phone}
	}
	return conv, nil
}

func (uc *OrderFlowUseCase) saveConversation(ctx context.Context, conv *whatsappDomain.Conversation) error {
	conv.UpdatedAt = time.Now()
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = conv.UpdatedAt
	}
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.conversationRepo.Upsert(ctx, conv)
	})
}

func (uc *OrderFlowUseCase) placeOrder(ctx context.Context, conv *whatsappDomain.Conversation, customer *customersDomain.Customer, phone string) (string, bool, error) {
	warehouseID, err := uc.resolveWarehouse(ctx)
	if err != nil {
		return "", false, fmt.Errorf("orderFlow: %w", err)
	}

	var customerID *string
	var email, addressLine string
	if customer != nil {
		id := customer.ID
		customerID = &id
		email = customer.Email
		addressLine = shippingAddressLine(*customer)
	} else {
		firstName, lastName := splitName(conv.PendingCustomerName)
		newCustomer, err := uc.createCustomer.Execute(ctx, appcustomers.CreateCustomerInput{
			FirstName: firstName, LastName: lastName, Email: conv.PendingCustomerEmail, PhoneE164: phone,
			Billing:  customersDomain.Address{FirstName: firstName, LastName: lastName, Email: conv.PendingCustomerEmail, Address1: conv.PendingCustomerAddress, Phone: phone},
			Shipping: customersDomain.Address{FirstName: firstName, LastName: lastName, Address1: conv.PendingCustomerAddress, Phone: phone},
		})
		if err != nil {
			return "", false, fmt.Errorf("orderFlow: create customer: %w", err)
		}
		customerID = &newCustomer.ID
		email = conv.PendingCustomerEmail
		addressLine = conv.PendingCustomerAddress
	}

	lineItems := make([]appoms.CreateSalesOrderLineItemInput, len(conv.Cart))
	var total int64
	currency := ""
	for i, item := range conv.Cart {
		subtotal := item.Quantity * item.PriceMinorUnits
		lineItems[i] = appoms.CreateSalesOrderLineItemInput{
			SKUID: item.SKUID, Name: item.ProductName, SKUCode: item.SKUCode, Quantity: item.Quantity,
			PriceMinorUnits: item.PriceMinorUnits, SubtotalMinorUnits: subtotal, TotalMinorUnits: subtotal,
		}
		total += subtotal
		currency = item.CurrencyCode
	}

	orderNumber := "WA-" + strings.ToUpper(idgen.NewV4()[:8])
	result, err := uc.createSalesOrder.Execute(ctx, appoms.CreateSalesOrderInput{
		Source: omsDomain.SalesOrderSourceWhatsApp, Number: orderNumber, OrderKey: orderNumber,
		Status: omsDomain.SalesOrderStatusPending, CustomerID: customerID,
		WarehouseID:     warehouseID,
		Billing:         buildOMSAddress(email, addressLine, phone),
		Shipping:        buildOMSAddress("", addressLine, phone),
		CurrencyCode:    currency,
		TotalMinorUnits: total,
		OccurredAt:      time.Now(),
		LineItems:       lineItems,
	})
	if err != nil {
		return "", false, fmt.Errorf("orderFlow: create sales order: %w", err)
	}

	conv.State = whatsappDomain.ConversationStateIdle
	conv.Cart = nil
	conv.PendingCustomerName, conv.PendingCustomerEmail, conv.PendingCustomerAddress = "", "", ""
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}

	// Payment link generation is deliberately best-effort and happens only
	// after the order is placed and the conversation is already cleared+
	// saved above: an error here must never propagate (which would make the
	// whole inbound message dispatch fail and redeliver -- CreateSalesOrderUseCase
	// has no idempotency key of its own, so replaying this handler would
	// place a SECOND order for the same YES). No payment method configured
	// is the normal, expected case for a tenant that hasn't set one up --
	// not an error.
	reply := fmt.Sprintf("Order #%s placed! Total: %s. We'll be in touch shortly about payment.",
		result.SalesOrder.Number, formatMoney(total, currency))
	if uc.generatePaymentLink != nil {
		link, ok, linkErr := uc.generatePaymentLink.Execute(ctx, total, currency, "Order #"+result.SalesOrder.Number, result.SalesOrder.Number)
		if linkErr != nil {
			log.Printf("orderFlow: generate payment link for order %s: %v", result.SalesOrder.Number, linkErr)
		} else if ok {
			reply = fmt.Sprintf("Order #%s placed! Total: %s.\nPay here: %s", result.SalesOrder.Number, formatMoney(total, currency), link.URL)
		}
	}
	return reply, true, nil
}
