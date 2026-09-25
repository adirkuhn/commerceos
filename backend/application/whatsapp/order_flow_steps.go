package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

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
