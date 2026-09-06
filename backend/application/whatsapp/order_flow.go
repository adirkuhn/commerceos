package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

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
	db               *gorm.DB
	conversationRepo whatsappDomain.ConversationRepository
	lookupProduct    productInfoLookup
	createSalesOrder salesOrderCreator
	createCustomer   customerCreator
}

func NewOrderFlowUseCase(
	db *gorm.DB,
	conversationRepo whatsappDomain.ConversationRepository,
	lookupProduct productInfoLookup,
	createSalesOrder salesOrderCreator,
	createCustomer customerCreator,
) *OrderFlowUseCase {
	return &OrderFlowUseCase{
		db: db, conversationRepo: conversationRepo, lookupProduct: lookupProduct,
		createSalesOrder: createSalesOrder, createCustomer: createCustomer,
	}
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

func (uc *OrderFlowUseCase) handleCancel(ctx context.Context, conv *whatsappDomain.Conversation) (string, bool, error) {
	hadSomethingToCancel := conv.State != whatsappDomain.ConversationStateIdle || len(conv.Cart) > 0
	conv.State = whatsappDomain.ConversationStateIdle
	conv.Cart = nil
	conv.PendingCustomerName, conv.PendingCustomerEmail, conv.PendingCustomerAddress = "", "", ""
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}
	if hadSomethingToCancel {
		return "Order cancelled. Your cart is now empty.", true, nil
	}
	return "Nothing to cancel.", true, nil
}

func (uc *OrderFlowUseCase) handleAddToCart(ctx context.Context, conv *whatsappDomain.Conversation, fields []string) (string, bool, error) {
	if len(fields) != 3 {
		return "To add a product, send: ORDER <code> <quantity>", true, nil
	}
	skuCode := fields[1]
	qty, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || qty <= 0 {
		return "Quantity must be a positive number. Send: ORDER <code> <quantity>", true, nil
	}

	info, err := uc.lookupProduct.Execute(ctx, skuCode)
	switch {
	case err == nil:
		// fall through below
	case errors.Is(err, ErrProductNotFound):
		return fmt.Sprintf("We couldn't find a product with code %s.", skuCode), true, nil
	case errors.Is(err, ErrNoWhatsAppChannel):
		return "WhatsApp ordering isn't set up for this account yet.", true, nil
	default:
		return "", false, fmt.Errorf("orderFlow: lookup product: %w", err)
	}
	if qty > info.StockQuantity {
		return fmt.Sprintf("Only %d in stock for %s.", info.StockQuantity, info.ProductName), true, nil
	}

	unitPrice := info.PriceMinorUnits
	if info.SalePriceMinorUnits != nil {
		unitPrice = *info.SalePriceMinorUnits
	}
	added := false
	for i := range conv.Cart {
		if conv.Cart[i].SKUID == info.SKUID {
			conv.Cart[i].Quantity += qty
			added = true
			break
		}
	}
	if !added {
		conv.Cart = append(conv.Cart, whatsappDomain.CartItem{
			SKUID: info.SKUID, SKUCode: info.SKUCode, ProductName: info.ProductName,
			Quantity: qty, PriceMinorUnits: unitPrice, CurrencyCode: info.CurrencyCode,
		})
	}
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}
	return fmt.Sprintf("Added %dx %s to your cart. Send CART to view it, or CHECKOUT to place your order.", qty, info.ProductName), true, nil
}

func (uc *OrderFlowUseCase) handleShowCart(conv *whatsappDomain.Conversation) (string, bool, error) {
	if len(conv.Cart) == 0 {
		return "Your cart is empty. Send ORDER <code> <quantity> to add a product.", true, nil
	}
	return cartSummary(conv.Cart) + "\nSend CHECKOUT to place your order, or CANCEL to clear your cart.", true, nil
}

func (uc *OrderFlowUseCase) handleCheckout(ctx context.Context, conv *whatsappDomain.Conversation, customer *customersDomain.Customer) (string, bool, error) {
	if len(conv.Cart) == 0 {
		return "Your cart is empty. Send ORDER <code> <quantity> to add a product first.", true, nil
	}
	if customerProfileComplete(customer) {
		conv.State = whatsappDomain.ConversationStateAwaitingOrderConfirmation
		if err := uc.saveConversation(ctx, conv); err != nil {
			return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
		}
		return confirmationSummary(conv.Cart, customer.FirstName+" "+customer.LastName, customer.Email, shippingAddressLine(*customer)), true, nil
	}
	conv.State = whatsappDomain.ConversationStateAwaitingNewCustomerName
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}
	return "Let's get your order set up! What's your full name?", true, nil
}

func (uc *OrderFlowUseCase) handleCaptureName(ctx context.Context, conv *whatsappDomain.Conversation, text string) (string, bool, error) {
	name := strings.TrimSpace(text)
	if name == "" {
		return "Please send your full name.", true, nil
	}
	conv.PendingCustomerName = name
	conv.State = whatsappDomain.ConversationStateAwaitingNewCustomerEmail
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}
	return "Thanks! What's your email address?", true, nil
}

func (uc *OrderFlowUseCase) handleCaptureEmail(ctx context.Context, conv *whatsappDomain.Conversation, text string) (string, bool, error) {
	email := strings.TrimSpace(text)
	if !strings.Contains(email, "@") {
		return "That doesn't look like a valid email. Please send a valid email address.", true, nil
	}
	conv.PendingCustomerEmail = email
	conv.State = whatsappDomain.ConversationStateAwaitingNewCustomerAddress
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}
	return "Great! What's your shipping address? (street, city, postcode, country)", true, nil
}

func (uc *OrderFlowUseCase) handleCaptureAddress(ctx context.Context, conv *whatsappDomain.Conversation, text string) (string, bool, error) {
	address := strings.TrimSpace(text)
	if address == "" {
		return "Please send your shipping address.", true, nil
	}
	conv.PendingCustomerAddress = address
	conv.State = whatsappDomain.ConversationStateAwaitingOrderConfirmation
	if err := uc.saveConversation(ctx, conv); err != nil {
		return "", false, fmt.Errorf("orderFlow: save conversation: %w", err)
	}
	return confirmationSummary(conv.Cart, conv.PendingCustomerName, conv.PendingCustomerEmail, conv.PendingCustomerAddress), true, nil
}

func (uc *OrderFlowUseCase) handleConfirmation(ctx context.Context, conv *whatsappDomain.Conversation, customer *customersDomain.Customer, phone, text string) (string, bool, error) {
	answer := strings.ToUpper(strings.TrimSpace(text))
	switch answer {
	case "YES", "Y":
		return uc.placeOrder(ctx, conv, customer, phone)
	case "NO", "N", "CANCEL":
		return uc.handleCancel(ctx, conv)
	default:
		return "Please reply YES to confirm your order, or NO to cancel.", true, nil
	}
}

func (uc *OrderFlowUseCase) placeOrder(ctx context.Context, conv *whatsappDomain.Conversation, customer *customersDomain.Customer, phone string) (string, bool, error) {
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

	return fmt.Sprintf("Order #%s placed! Total: %s. We'll be in touch shortly about payment.",
		result.SalesOrder.Number, formatMoney(total, currency)), true, nil
}
