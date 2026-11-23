package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/adirkuhn/commerceos/domain/assistant"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const maxQuoteLines = 100

type priceLister interface {
	Execute(ctx context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error)
}

type customerGetter interface {
	Execute(ctx context.Context, id string) (*customersDomain.Customer, error)
}

type quoteCreator interface {
	Execute(ctx context.Context, in quotes.Input) (*quotes.Quote, error)
}

// CreateQuoteAction proposes a new B2B quote, priced the way the admin's
// quote editor prices it; confirming saves it and offers its conversion.
type CreateQuoteAction struct {
	prices    priceLister
	customers customerGetter
	create    quoteCreator
	now       func() time.Time
}

func NewCreateQuoteAction(prices priceLister, customers customerGetter, create quoteCreator, now func() time.Time) *CreateQuoteAction {
	return &CreateQuoteAction{prices: prices, customers: customers, create: create, now: now}
}

var _ appassistant.ProposalAction = (*CreateQuoteAction)(nil)

func (a *CreateQuoteAction) Kind() assistant.ProposalKind { return assistant.KindCreateQuote }

func (a *CreateQuoteAction) Tool() assistant.ToolDefinition {
	line := object(map[string]any{
		"product_id": str("The product's id (from search or get_product)."),
		"sku_id":     str("The SKU's id (from get_product)."),
		"quantity":   map[string]any{"type": "integer", "minimum": 1, "description": "How many units."},
		"unit_price_minor_units": map[string]any{"type": "integer", "minimum": 0,
			"description": "Only when the user stated a price per unit, in cents. Leave out to use the shop's price with the discount."},
	}, "product_id", "sku_id", "quantity")
	return assistant.ToolDefinition{
		Name: "propose_quote",
		Description: "Propose a new B2B quote for the user to confirm. Look the products up first (search, get_product) " +
			"to get each product_id and sku_id, and the customer (search) for customer_id. Nothing is saved until the " +
			"user confirms; after this call, stop and tell them the proposal is ready.",
		Parameters: object(map[string]any{
			"name":             str("What the quote is called, usually the customer's or the request's name."),
			"customer_id":      str("The customer's id, when the quote is for a known customer."),
			"discount_percent": map[string]any{"type": "number", "minimum": 0, "maximum": 100, "description": "Discount off the shop's prices, in percent."},
			"lines":            map[string]any{"type": "array", "minItems": 1, "maxItems": maxQuoteLines, "items": line},
		}, "name", "lines"),
	}
}

type quoteArgs struct {
	Name            string          `json:"name"`
	CustomerID      string          `json:"customer_id"`
	DiscountPercent float64         `json:"discount_percent"`
	Lines           []quoteLineArgs `json:"lines"`
}

type quoteLineArgs struct {
	ProductID           string `json:"product_id"`
	SKUID               string `json:"sku_id"`
	Quantity            int64  `json:"quantity"`
	UnitPriceMinorUnits *int64 `json:"unit_price_minor_units"`
}

// quotePayload is quotes.Input as stored on the proposal.
type quotePayload struct {
	CustomerID          *string            `json:"customer_id"`
	Name                string             `json:"name"`
	DiscountBasisPoints int                `json:"discount_basis_points"`
	CurrencyCode        string             `json:"currency_code"`
	Lines               []quotePayloadLine `json:"lines"`
}

type quotePayloadLine struct {
	ProductID           string `json:"product_id"`
	SKUID               string `json:"sku_id"`
	Quantity            int64  `json:"quantity"`
	UnitPriceMinorUnits int64  `json:"unit_price_minor_units"`
}

type quotePreview struct {
	Name                string             `json:"name"`
	Customer            *customerPreview   `json:"customer"`
	DiscountBasisPoints int                `json:"discount_basis_points"`
	CurrencyCode        string             `json:"currency_code"`
	PricesIncludeTax    bool               `json:"prices_include_tax"`
	Lines               []quoteLinePreview `json:"lines"`
	TotalMinorUnits     int64              `json:"total_minor_units"`
}

type customerPreview struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type quoteLinePreview struct {
	ProductID           string `json:"product_id"`
	SKUID               string `json:"sku_id"`
	ProductName         string `json:"product_name"`
	VariantName         string `json:"variant_name"`
	SKUCode             string `json:"sku_code"`
	Quantity            int64  `json:"quantity"`
	ListPriceMinorUnits int64  `json:"list_price_minor_units"`
	UnitPriceMinorUnits int64  `json:"unit_price_minor_units"`
	LineTotalMinorUnits int64  `json:"line_total_minor_units"`
}

func (a *CreateQuoteAction) Prepare(ctx context.Context, args json.RawMessage) (json.RawMessage, json.RawMessage, error) {
	var in quoteArgs
	if err := decode(args, &in); err != nil {
		return nil, nil, err
	}
	if err := checkQuoteArgs(in); err != nil {
		return nil, nil, err
	}
	bp := discountBasisPoints(in.DiscountPercent)

	priced, err := a.prices.Execute(ctx, apppms.ListPriceListInput{Page: 1, PerPage: maxQuoteLines, ProductIDs: productIDs(in.Lines)})
	if err != nil {
		return nil, nil, fmt.Errorf("load prices: %w", err)
	}
	rows := make(map[string]apppms.PriceListRow, len(priced.Rows))
	for _, r := range priced.Rows {
		rows[r.SKUID] = r
	}

	payload := quotePayload{Name: in.Name, DiscountBasisPoints: bp}
	preview := quotePreview{Name: in.Name, DiscountBasisPoints: bp, PricesIncludeTax: priced.PricesIncludeTax}
	for i, l := range in.Lines {
		row, ok := rows[l.SKUID]
		if !ok || row.ProductID != l.ProductID {
			return nil, nil, fmt.Errorf("line %d: sku %s is not an active SKU of product %s -- check it with get_product", i+1, l.SKUID, l.ProductID)
		}
		if payload.CurrencyCode == "" {
			payload.CurrencyCode = row.CurrencyCode
		} else if row.CurrencyCode != payload.CurrencyCode {
			return nil, nil, fmt.Errorf("line %d: priced in %s while earlier lines are in %s -- a quote has one currency", i+1, row.CurrencyCode, payload.CurrencyCode)
		}
		list := row.PriceMinorUnits
		if row.SalePriceMinorUnits != nil {
			list = *row.SalePriceMinorUnits
		}
		unit := applyDiscount(list, bp)
		if l.UnitPriceMinorUnits != nil {
			unit = *l.UnitPriceMinorUnits
		}
		payload.Lines = append(payload.Lines, quotePayloadLine{ProductID: l.ProductID, SKUID: l.SKUID, Quantity: l.Quantity, UnitPriceMinorUnits: unit})
		preview.Lines = append(preview.Lines, quoteLinePreview{
			ProductID: l.ProductID, SKUID: l.SKUID, ProductName: row.ProductName, VariantName: row.VariantName, SKUCode: row.SKUCode,
			Quantity: l.Quantity, ListPriceMinorUnits: list, UnitPriceMinorUnits: unit, LineTotalMinorUnits: unit * l.Quantity,
		})
		preview.TotalMinorUnits += unit * l.Quantity
	}
	preview.CurrencyCode = payload.CurrencyCode

	if in.CustomerID != "" {
		c, err := a.customers.Execute(ctx, in.CustomerID)
		if err != nil {
			return nil, nil, fmt.Errorf("customer %s: %w -- look it up with search", in.CustomerID, err)
		}
		id := c.ID
		payload.CustomerID = &id
		preview.Customer = &customerPreview{ID: c.ID, Name: c.DisplayName(), Email: c.Email}
	}

	// Dry run of the domain's own rules (name, discount, currency, lines),
	// so a bad proposal is caught now rather than on confirm.
	if _, err := quotes.NewQuote(tenant.IDFromContext(ctx), payload.input(), tenant.UserIDFromContext(ctx), a.now()); err != nil {
		return nil, nil, fmt.Errorf("the quote isn't valid: %w", err)
	}

	payloadJSON, err := encode(payload)
	if err != nil {
		return nil, nil, err
	}
	previewJSON, err := encode(preview)
	if err != nil {
		return nil, nil, err
	}
	return payloadJSON, previewJSON, nil
}

func checkQuoteArgs(in quoteArgs) error {
	switch {
	case in.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidArguments)
	case len(in.Lines) == 0:
		return fmt.Errorf("%w: at least one line is required", ErrInvalidArguments)
	case len(in.Lines) > maxQuoteLines:
		return fmt.Errorf("%w: at most %d lines", ErrInvalidArguments, maxQuoteLines)
	case in.DiscountPercent < 0 || in.DiscountPercent > 100:
		return fmt.Errorf("%w: discount_percent must be between 0 and 100", ErrInvalidArguments)
	}
	for i, l := range in.Lines {
		if l.ProductID == "" || l.SKUID == "" {
			return fmt.Errorf("%w: line %d needs product_id and sku_id", ErrInvalidArguments, i+1)
		}
		if l.Quantity < 1 {
			return fmt.Errorf("%w: line %d quantity must be at least 1", ErrInvalidArguments, i+1)
		}
		if l.UnitPriceMinorUnits != nil && *l.UnitPriceMinorUnits < 0 {
			return fmt.Errorf("%w: line %d price can't be negative", ErrInvalidArguments, i+1)
		}
	}
	return nil
}

func productIDs(lines []quoteLineArgs) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range lines {
		if !seen[l.ProductID] {
			seen[l.ProductID] = true
			out = append(out, l.ProductID)
		}
	}
	return out
}

func (p quotePayload) input() quotes.Input {
	in := quotes.Input{CustomerID: p.CustomerID, Name: p.Name, DiscountBasisPoints: p.DiscountBasisPoints, CurrencyCode: p.CurrencyCode}
	for _, l := range p.Lines {
		in.Lines = append(in.Lines, quotes.Line{ProductID: l.ProductID, SKUID: l.SKUID, Quantity: l.Quantity, UnitPriceMinorUnits: l.UnitPriceMinorUnits})
	}
	return in
}

func (a *CreateQuoteAction) Execute(ctx context.Context, payload json.RawMessage, _ appassistant.ConfirmChoices) (appassistant.ActionResult, error) {
	var p quotePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return appassistant.ActionResult{}, fmt.Errorf("createQuoteAction: decode payload: %w", err)
	}
	q, err := a.create.Execute(ctx, p.input())
	if err != nil {
		return appassistant.ActionResult{}, fmt.Errorf("createQuoteAction: %w", err)
	}
	followUp, err := encode(map[string]string{"quote_id": q.ID})
	if err != nil {
		return appassistant.ActionResult{}, err
	}
	return appassistant.ActionResult{
		ID: q.ID, Ref: quotes.Reference(q.Number),
		FollowUp: &appassistant.FollowUp{Kind: assistant.KindConvertQuote, Args: followUp},
	}, nil
}
