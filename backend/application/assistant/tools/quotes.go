package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

// GetQuoteTool reads one saved B2B quote by id or by its reference.
type GetQuoteTool struct {
	get  quoteGetter
	list quoteLister
}

func NewGetQuoteTool(get quoteGetter, list quoteLister) *GetQuoteTool {
	return &GetQuoteTool{get: get, list: list}
}

func (t *GetQuoteTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_quote",
		Description: "Read one saved B2B quote -- customer, discount, lines and total -- by its id or its reference as the " +
			"shop shows it (Q-0012, or just 12). To find a quote by name or customer, use list_quotes.",
		Parameters: object(map[string]any{"quote": str("The quote id (uuid) or reference, e.g. Q-0012.")}, "quote"),
	}
}

var quoteReference = regexp.MustCompile(`^(?i)#?\s*q?-?\s*0*(\d+)$`)

// id resolves a reference through list_quotes' search, accepting only the
// exact number -- search also matches names and other numbers.
func (t *GetQuoteTool) id(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if uuidPattern.MatchString(ref) {
		return ref, nil
	}
	m := quoteReference.FindStringSubmatch(ref)
	if m == nil {
		return "", fmt.Errorf("%w: %q is neither a quote id nor a reference like Q-0012 -- use list_quotes to search by name", ErrInvalidArguments, ref)
	}
	number, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return "", fmt.Errorf("%w: quote number %q", ErrInvalidArguments, m[1])
	}
	res, err := t.list.Execute(ctx, appquotes.ListQuotesInput{Page: 1, PerPage: maxLimit, Search: m[1]})
	if err != nil {
		return "", err
	}
	var near []string
	for _, q := range res.Quotes {
		if q.Number == number {
			return q.ID, nil
		}
		near = append(near, quotes.Reference(q.Number))
	}
	if len(near) == 0 {
		return "", fmt.Errorf("no quote %s", quotes.Reference(number))
	}
	return "", fmt.Errorf("no quote %s; similar: %s", quotes.Reference(number), strings.Join(near, ", "))
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
		Quote string `json:"quote"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("quote", args.Quote); err != nil {
		return "", err
	}
	id, err := t.id(ctx, args.Quote)
	if err != nil {
		return "", err
	}
	q, err := t.get.Execute(ctx, id)
	if err != nil {
		return "", err
	}
	view := struct {
		ID                  string             `json:"id"`
		Number              int64              `json:"number"`
		Reference           string             `json:"reference"`
		Name                string             `json:"name"`
		Customer            *quoteCustomerView `json:"customer,omitempty"`
		CurrencyCode        string             `json:"currency_code"`
		DiscountBasisPoints int                `json:"discount_basis_points"`
		Lines               []quoteLineView    `json:"lines"`
		TotalMinorUnits     int64              `json:"total_minor_units"`
		CreatedAt           string             `json:"created_at"`
		UpdatedAt           string             `json:"updated_at"`
	}{
		ID: q.ID, Number: q.Number, Reference: quotes.Reference(q.Number), Name: q.Name, CurrencyCode: q.CurrencyCode, DiscountBasisPoints: q.DiscountBasisPoints,
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
