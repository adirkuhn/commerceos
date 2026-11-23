package tools

import (
	"context"
	"encoding/json"
	"strings"

	appsearch "github.com/adirkuhn/commerceos/application/search"
	"github.com/adirkuhn/commerceos/domain/assistant"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type globalSearcher interface {
	Execute(ctx context.Context, query string) (*appsearch.GlobalSearchResult, error)
}

// SearchTool finds products, customers, suppliers and orders by name,
// code, number or email -- usually the model's first step.
type SearchTool struct{ search globalSearcher }

func NewSearchTool(search globalSearcher) *SearchTool { return &SearchTool{search: search} }

func (t *SearchTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "search",
		Description: "Fuzzy search (accent-insensitive) across products, customers, suppliers and sales orders by name, " +
			"SKU/code, order number or email; returns ids to use with the other tools. Up to 5 hits per kind.",
		Parameters: object(map[string]any{"query": str("What to look for, e.g. a product name, customer name or order number.")}, "query"),
	}
}

type searchArgs struct {
	Query string `json:"query"`
}

type searchView struct {
	Products    []productHit  `json:"products"`
	Customers   []customerHit `json:"customers"`
	Suppliers   []supplierHit `json:"suppliers"`
	SalesOrders []orderHit    `json:"sales_orders"`
}

type productHit struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type customerHit struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CompanyName  string `json:"company_name,omitempty"`
	Email        string `json:"email,omitempty"`
	ClientNumber string `json:"client_number,omitempty"`
}

type supplierHit struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type orderHit struct {
	ID              string `json:"id"`
	Number          string `json:"number"`
	Status          string `json:"status"`
	CustomerName    string `json:"customer_name,omitempty"`
	TotalMinorUnits int64  `json:"total_minor_units"`
	CurrencyCode    string `json:"currency_code"`
	OccurredAt      string `json:"occurred_at"`
}

func (t *SearchTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args searchArgs
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("query", args.Query); err != nil {
		return "", err
	}
	res, err := t.search.Execute(ctx, args.Query)
	if err != nil {
		return "", err
	}
	view := searchView{Products: []productHit{}, Customers: []customerHit{}, Suppliers: []supplierHit{}, SalesOrders: []orderHit{}}
	for _, p := range res.Products {
		view.Products = append(view.Products, productHit{ID: p.ID, Name: p.Name, Status: string(p.Status)})
	}
	for _, c := range res.Customers {
		view.Customers = append(view.Customers, customerHit{
			ID: c.ID, Name: fullName(c.FirstName, c.LastName), CompanyName: c.CompanyName, Email: c.Email, ClientNumber: c.ClientNumber,
		})
	}
	for _, s := range res.Suppliers {
		view.Suppliers = append(view.Suppliers, supplierHit{ID: s.ID, Name: s.Name, Email: s.Email})
	}
	for _, o := range res.SalesOrders {
		view.SalesOrders = append(view.SalesOrders, toOrderHit(o))
	}
	return toJSON(view)
}

func toOrderHit(o omsDomain.SalesOrder) orderHit {
	return orderHit{
		ID: o.ID, Number: o.Number, Status: string(o.Status), CustomerName: orderCustomerName(o),
		TotalMinorUnits: o.TotalMinorUnits, CurrencyCode: o.CurrencyCode, OccurredAt: formatTime(o.OccurredAt),
	}
}

func orderCustomerName(o omsDomain.SalesOrder) string {
	if name := fullName(o.Billing.FirstName, o.Billing.LastName); name != "" {
		return name
	}
	return o.Billing.Company
}

func fullName(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}
