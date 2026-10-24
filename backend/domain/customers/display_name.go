package customers

import "strings"

// DisplayName is how a customer is named in pickers and on documents: the
// person's name, else the billing company, else the email.
func (c Customer) DisplayName() string {
	if name := strings.TrimSpace(strings.TrimSpace(c.FirstName) + " " + strings.TrimSpace(c.LastName)); name != "" {
		return name
	}
	if company := strings.TrimSpace(c.Billing.Company); company != "" {
		return company
	}
	return c.Email
}

// Identifiable is whether there's anything to tell this customer apart by:
// a name, company or email (DisplayName), or a phone number. Email is
// optional -- walk-in and B2B customers often have none.
func (c Customer) Identifiable() bool {
	return strings.TrimSpace(c.DisplayName()) != "" || c.PhoneE164 != "" || strings.TrimSpace(c.Billing.Phone) != ""
}
