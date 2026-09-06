package whatsapp

import (
	"fmt"
	"strings"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

// customerProfileComplete reports whether an already-resolved customer has
// enough on file to skip straight to order confirmation -- an email and at
// least one address (shipping preferred, billing as fallback). A nil
// customer, or one missing either, goes through new-customer capture
// instead (see OrderFlowUseCase.handleCheckout) -- deliberately never a
// partial "ask only what's missing" flow: distinguishing "known customer
// with a gap" from "brand new customer" would need an update-in-place path
// this phase doesn't build (see the WhatsApp integration project memory
// for why that was cut from scope).
func customerProfileComplete(customer *customersDomain.Customer) bool {
	if customer == nil || customer.Email == "" {
		return false
	}
	return customer.Shipping.Address1 != "" || customer.Billing.Address1 != ""
}

// shippingAddressLine renders a customer's best-available address as one
// line for a chat confirmation summary -- shipping preferred, billing
// fallback (mirrors customerProfileComplete's own preference order).
func shippingAddressLine(customer customersDomain.Customer) string {
	if customer.Shipping.Address1 != "" {
		return customer.Shipping.Address1
	}
	return customer.Billing.Address1
}

// splitName divides a free-typed full name into first/last for Customer's
// separate fields -- first token is FirstName, everything else is
// LastName; a single-token name gets an empty LastName rather than
// guessing.
func splitName(fullName string) (first, last string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// buildOMSAddress builds the minimal oms.Address a WhatsApp order snapshots
// -- only Address1/Email/Phone are ever known from the chat flow (no
// separate city/postcode/country capture in this MVP), everything else
// stays zero-value. email is passed "" for Shipping, matching Address's own
// doc comment ("Email is only meaningful on Billing").
func buildOMSAddress(email, addressLine, phone string) omsDomain.Address {
	return omsDomain.Address{Address1: addressLine, Email: email, Phone: phone}
}

// formatMoney renders a minor-units amount as "12.34 EUR" -- same
// convention (float64(minor)/100) this codebase already uses for display
// (see application/channels' Bol push, application/whatsapp's own
// formatProductInfo).
func formatMoney(minorUnits int64, currencyCode string) string {
	return fmt.Sprintf("%.2f %s", float64(minorUnits)/100, currencyCode)
}

// cartSummary renders a Conversation's cart as multi-line text: one line
// per item, then a total line.
func cartSummary(cart []whatsappDomain.CartItem) string {
	var b strings.Builder
	b.WriteString("Your cart:\n")
	var total int64
	for _, item := range cart {
		subtotal := item.Quantity * item.PriceMinorUnits
		total += subtotal
		fmt.Fprintf(&b, "- %dx %s (%s) = %s\n", item.Quantity, item.ProductName, item.SKUCode, formatMoney(subtotal, item.CurrencyCode))
	}
	currency := ""
	if len(cart) > 0 {
		currency = cart[0].CurrencyCode
	}
	fmt.Fprintf(&b, "Total: %s", formatMoney(total, currency))
	return b.String()
}

// confirmationSummary renders the final "here's what we have, reply YES/NO"
// message -- shared by both checkout paths (already-known customer straight
// from handleCheckout, new-customer capture ending in handleCaptureAddress).
func confirmationSummary(cart []whatsappDomain.CartItem, name, email, addressLine string) string {
	return fmt.Sprintf("%s\n\nShip to: %s\nName: %s\nEmail: %s\n\nReply YES to confirm your order, or NO to cancel.",
		cartSummary(cart), addressLine, name, email)
}
