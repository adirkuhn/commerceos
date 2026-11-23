package tools

import (
	"context"
	"encoding/json"

	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

type quoteGetter interface {
	Execute(ctx context.Context, id string) (*quotes.Quote, error)
}

type quoteLister interface {
	Execute(ctx context.Context, in appquotes.ListQuotesInput) (*appquotes.ListQuotesResult, error)
}

// GetQuoteTool reads one saved B2B quote.
type GetQuoteTool struct{ get quoteGetter }

func NewGetQuoteTool(get quoteGetter) *GetQuoteTool { return &GetQuoteTool{get: get} }

func (t *GetQuoteTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name:        "get_quote",
		Description: "Read one saved B2B quote -- customer, discount, lines and total -- by its id (from list_quotes).",
		Parameters:  object(map[string]any{"quote_id": str("The quote id.")}, "quote_id"),
	}
}

type quoteCustomerView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type quoteLineView struct {
	ProductID           string `json:"product_id"`
	SKUID               string `json:"sku_id"`
	Quantity            int64  `json:"quantity"`
	UnitPriceMinorUnits int64  `json:"unit_price_minor_units"`
}

func (t *GetQuoteTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		QuoteID string `json:"quote_id"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("quote_id", args.QuoteID); err != nil {
		return "", err
	}
	q, err := t.get.Execute(ctx, args.QuoteID)
	if err != nil {
		return "", err
	}
	view := struct {
		ID                  string             `json:"id"`
		Number              int64              `json:"number"`
		Name                string             `json:"name"`
		Customer            *quoteCustomerView `json:"customer,omitempty"`
		CurrencyCode        string             `json:"currency_code"`
		DiscountBasisPoints int                `json:"discount_basis_points"`
		Lines               []quoteLineView    `json:"lines"`
		TotalMinorUnits     int64              `json:"total_minor_units"`
		CreatedAt           string             `json:"created_at"`
		UpdatedAt           string             `json:"updated_at"`
	}{
		ID: q.ID, Number: q.Number, Name: q.Name, CurrencyCode: q.CurrencyCode, DiscountBasisPoints: q.DiscountBasisPoints,
		Lines: make([]quoteLineView, 0, len(q.Lines)), TotalMinorUnits: q.Total(),
		CreatedAt: formatTime(q.CreatedAt), UpdatedAt: formatTime(q.UpdatedAt),
	}
	if q.Customer != nil {
		view.Customer = &quoteCustomerView{ID: q.Customer.ID, Name: q.Customer.Name, Email: q.Customer.Email}
	}
	for _, l := range q.Lines {
		view.Lines = append(view.Lines, quoteLineView{ProductID: l.ProductID, SKUID: l.SKUID, Quantity: l.Quantity, UnitPriceMinorUnits: l.UnitPriceMinorUnits})
	}
	return toJSON(view)
}

// ListQuotesTool lists saved quotes, most recently saved first.
type ListQuotesTool struct{ list quoteLister }

func NewListQuotesTool(list quoteLister) *ListQuotesTool { return &ListQuotesTool{list: list} }

func (t *ListQuotesTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name:        "list_quotes",
		Description: "List saved B2B quotes, most recently saved first, optionally matching a search text (name or customer).",
		Parameters:  object(map[string]any{"search": str("Optional text to match."), "limit": limitProp()}),
	}
}

type quoteHit struct {
	ID              string `json:"id"`
	Number          int64  `json:"number"`
	Name            string `json:"name"`
	CustomerName    string `json:"customer_name,omitempty"`
	LineCount       int    `json:"line_count"`
	TotalMinorUnits int64  `json:"total_minor_units"`
	CurrencyCode    string `json:"currency_code"`
	UpdatedAt       string `json:"updated_at"`
}

func (t *ListQuotesTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Search string `json:"search"`
		Limit  int    `json:"limit"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	res, err := t.list.Execute(ctx, appquotes.ListQuotesInput{Page: 1, PerPage: clampLimit(args.Limit), Search: args.Search})
	if err != nil {
		return "", err
	}
	out := struct {
		Total  int64      `json:"total"`
		Quotes []quoteHit `json:"quotes"`
	}{Total: res.Total, Quotes: make([]quoteHit, 0, len(res.Quotes))}
	for _, q := range res.Quotes {
		hit := quoteHit{ID: q.ID, Number: q.Number, Name: q.Name, LineCount: q.LineCount, TotalMinorUnits: q.TotalMinorUnits,
			CurrencyCode: q.CurrencyCode, UpdatedAt: formatTime(q.UpdatedAt)}
		if q.Customer != nil {
			hit.CustomerName = q.Customer.Name
		}
		out.Quotes = append(out.Quotes, hit)
	}
	return toJSON(out)
}
