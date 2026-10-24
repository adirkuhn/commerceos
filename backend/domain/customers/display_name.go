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
