package tools

import (
	"context"
	"encoding/json"
	"strings"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/domain/assistant"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type customersLister interface {
	Execute(ctx context.Context) ([]customersDomain.Customer, error)
}

type customerGetter interface {
	Execute(ctx context.Context, id string) (*customersDomain.Customer, error)
}

type contactPersonsLister interface {
	Execute(ctx context.Context, customerID string) ([]customersDomain.ContactPerson, error)
}

// SearchCustomersTool finds customers by name, company, email, phone,
// client number or VAT number -- broader than search, which matches names.
type SearchCustomersTool struct{ list customersLister }

func NewSearchCustomersTool(list customersLister) *SearchCustomersTool {
	return &SearchCustomersTool{list: list}
}

func (t *SearchCustomersTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "search_customers",
		Description: "Find customers by part of their name, company, email, phone number, client number or VAT number " +
			"(accents and case ignored). Use get_customer with the id for details.",
		Parameters: object(map[string]any{
			"query": str("What to look for, e.g. a name, an email or a phone number."),
			"limit": limitProp(),
		}, "query"),
	}
}

type customerHitView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	ClientNumber string `json:"client_number,omitempty"`
	City         string `json:"city,omitempty"`
}

func (t *SearchCustomersTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("query", args.Query); err != nil {
		return "", err
	}
	all, err := t.list.Execute(ctx)
	if err != nil {
		return "", err
	}
	limit := clampLimit(args.Limit)
	out := struct {
		TotalMatches int               `json:"total_matches"`
		Customers    []customerHitView `json:"customers"`
	}{Customers: []customerHitView{}}
	for _, c := range all {
		if !customerMatches(c, args.Query) {
			continue
		}
		out.TotalMatches++
		if len(out.Customers) < limit {
			out.Customers = append(out.Customers, customerHitView{
				ID: c.ID, Name: c.DisplayName(), Type: string(c.Type), Email: c.Email, Phone: c.PhoneE164,
				ClientNumber: c.ClientNumber, City: c.Billing.City,
			})
		}
	}
	return toJSON(out)
}

func customerMatches(c customersDomain.Customer, query string) bool {
	q := fold(query)
	for _, field := range []string{c.DisplayName(), c.CompanyName, c.Billing.Company, fullName(c.FirstName, c.LastName),
		c.Email, c.Billing.Email, c.ClientNumber, c.VATNumber, c.KVKNumber} {
		if field != "" && strings.Contains(fold(field), q) {
			return true
		}
	}
	// Phones: "0612345678" finds "+31612345678" -- compare the digits
	// without the leading zero or country code.
	if d := strings.TrimLeft(digits(query), "0"); len(d) >= 6 {
		for _, phone := range []string{c.PhoneE164, c.Billing.Phone} {
			if strings.Contains(digits(phone), d) {
				return true
			}
		}
	}
	return false
}

// accentFolder covers the accents of the shop's languages (Portuguese,
// Dutch, the usual Latin ones) -- enough for matching typed names.
var accentFolder = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n", "ĳ", "ij",
)

func fold(s string) string { return accentFolder.Replace(strings.ToLower(strings.TrimSpace(s))) }

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// GetCustomerTool reads one customer with contact persons and how much
// they've ordered.
type GetCustomerTool struct {
	get     customerGetter
	persons contactPersonsLister
	orders  salesOrderLister
}

func NewGetCustomerTool(get customerGetter, persons contactPersonsLister, orders salesOrderLister) *GetCustomerTool {
	return &GetCustomerTool{get: get, persons: persons, orders: orders}
}

func (t *GetCustomerTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_customer",
		Description: "Read one customer (id from search or search_customers): type, company and contact details, VAT and " +
			"chamber of commerce numbers, addresses, note, contact persons, number of orders and the latest order.",
		Parameters: object(map[string]any{"customer_id": str("The customer id.")}, "customer_id"),
	}
}

type contactPersonView struct {
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

type lastOrderView struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Status string `json:"status"`
	Date   string `json:"date"`
}

type customerView struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Type            string              `json:"type"`
	CompanyName     string              `json:"company_name,omitempty"`
	ContactName     string              `json:"contact_name,omitempty"`
	Email           string              `json:"email,omitempty"`
	Phone           string              `json:"phone,omitempty"`
	ClientNumber    string              `json:"client_number,omitempty"`
	VATNumber       string              `json:"vat_number,omitempty"`
	KVKNumber       string              `json:"kvk_number,omitempty"`
	BillingAddress  string              `json:"billing_address,omitempty"`
	ShippingAddress string              `json:"shipping_address,omitempty"`
	Note            string              `json:"note,omitempty"`
	PayingCustomer  bool                `json:"paying_customer"`
	ContactPersons  []contactPersonView `json:"contact_persons"`
	OrderCount      int64               `json:"order_count"`
	LastOrder       *lastOrderView      `json:"last_order,omitempty"`
	CreatedAt       string              `json:"created_at"`
}

func (t *GetCustomerTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		CustomerID string `json:"customer_id"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("customer_id", args.CustomerID); err != nil {
		return "", err
	}
	c, err := t.get.Execute(ctx, args.CustomerID)
	if err != nil {
		return "", err
	}
	persons, err := t.persons.Execute(ctx, c.ID)
	if err != nil {
		return "", err
	}
	id := c.ID
	orders, err := t.orders.Execute(ctx, appoms.ListSalesOrdersInput{
		Page: 1, PerPage: 1, CustomerID: &id, SortBy: omsDomain.SalesOrderSortOccurredAt, SortDir: omsDomain.SortDesc,
	})
	if err != nil {
		return "", err
	}
	view := customerView{
		ID: c.ID, Name: c.DisplayName(), Type: string(c.Type), CompanyName: c.CompanyName, ContactName: fullName(c.FirstName, c.LastName),
		Email: c.Email, Phone: firstNonEmpty(c.PhoneE164, c.Billing.Phone), ClientNumber: c.ClientNumber, VATNumber: c.VATNumber,
		KVKNumber: c.KVKNumber, BillingAddress: customerAddress(c.Billing), ShippingAddress: customerAddress(c.Shipping),
		Note: c.CustomerNote, PayingCustomer: c.IsPayingCustomer, ContactPersons: []contactPersonView{},
		OrderCount: orders.Total, CreatedAt: formatTime(c.DateCreated),
	}
	for _, p := range persons {
		view.ContactPersons = append(view.ContactPersons, contactPersonView{Name: fullName(p.FirstName, p.LastName), Role: p.Role, Email: p.Email, Phone: p.Phone})
	}
	if len(orders.SalesOrders) > 0 {
		o := orders.SalesOrders[0]
		view.LastOrder = &lastOrderView{ID: o.ID, Number: o.Number, Status: string(o.Status), Date: formatTime(o.OccurredAt)}
	}
	return toJSON(view)
}

func customerAddress(a customersDomain.Address) string {
	var out []string
	for _, p := range []string{strings.TrimSpace(a.Address1 + " " + a.Address2), strings.TrimSpace(a.Postcode + " " + a.City), a.Country} {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}
