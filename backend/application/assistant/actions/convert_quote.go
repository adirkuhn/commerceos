package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/assistant"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

// ErrNoWarehouse: the shop has no warehouse to fulfil the order from.
var ErrNoWarehouse = errors.New("the shop has no warehouse to create the order in")

// ErrEmptyQuote: a quote without lines has nothing to order.
var ErrEmptyQuote = errors.New("the quote has no lines")

type quoteGetter interface {
	Execute(ctx context.Context, id string) (*quotes.Quote, error)
}

type warehouseLister interface {
	Execute(ctx context.Context) ([]invDomain.Warehouse, error)
}

type quoteStockChecker interface {
	Execute(ctx context.Context, quoteID, warehouseID string) (*appquotes.StockCheck, error)
}

type quoteConverter interface {
	Execute(ctx context.Context, in appquotes.ConvertQuoteInput) (*appquotes.ConvertQuoteResult, error)
}

// ConvertQuoteAction proposes turning a saved quote into a sales order,
// from the first warehouse like the admin's convert dialog; the person
// can pick another warehouse when confirming.
type ConvertQuoteAction struct {
	prices     priceLister
	quotes     quoteGetter
	warehouses warehouseLister
	stock      quoteStockChecker
	convert    quoteConverter
}

func NewConvertQuoteAction(prices priceLister, quotes quoteGetter, warehouses warehouseLister, stock quoteStockChecker, convert quoteConverter) *ConvertQuoteAction {
	return &ConvertQuoteAction{prices: prices, quotes: quotes, warehouses: warehouses, stock: stock, convert: convert}
}

var _ appassistant.ProposalAction = (*ConvertQuoteAction)(nil)

func (a *ConvertQuoteAction) Kind() assistant.ProposalKind { return assistant.KindConvertQuote }

func (a *ConvertQuoteAction) Tool() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "propose_quote_conversion",
		Description: "Propose turning a saved quote into a sales order, for the user to confirm. Needs the quote's id " +
			"(find it with list_quotes or get_quote). Nothing is created until the user confirms; after this call, stop " +
			"and tell them the proposal is ready.",
		Parameters: object(map[string]any{"quote_id": str("The saved quote's id.")}, "quote_id"),
	}
}

type convertArgs struct {
	QuoteID string `json:"quote_id"`
}

type convertPayload struct {
	QuoteID     string `json:"quote_id"`
	WarehouseID string `json:"warehouse_id"`
}

type convertPreview struct {
	QuoteID         string             `json:"quote_id"`
	QuoteRef        string             `json:"quote_ref"`
	Name            string             `json:"name"`
	Customer        *customerPreview   `json:"customer"`
	LineCount       int                `json:"line_count"`
	TotalMinorUnits int64              `json:"total_minor_units"`
	CurrencyCode    string             `json:"currency_code"`
	WarehouseID     string             `json:"warehouse_id"`
	Warehouses      []warehousePreview `json:"warehouses"`
	// Lines is what becomes the order: the quote's stored lines and prices.
	Lines     []convertLinePreview `json:"lines"`
	Shortfall []shortfallPreview   `json:"shortfall"`
}

// skuLabel names a SKU for a person; empty strings when the SKU is no
// longer in the price list (archived or deleted since the quote).
type skuLabel struct {
	ProductName string `json:"product_name"`
	VariantName string `json:"variant_name"`
	SKUCode     string `json:"sku_code"`
}

type convertLinePreview struct {
	ProductID string `json:"product_id"`
	SKUID     string `json:"sku_id"`
	skuLabel
	Quantity            int64 `json:"quantity"`
	UnitPriceMinorUnits int64 `json:"unit_price_minor_units"`
	LineTotalMinorUnits int64 `json:"line_total_minor_units"`
}

type warehousePreview struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// shortfallPreview is a quote line the default warehouse can't fully
// supply (the order can still be created; it's flagged for stock review).
type shortfallPreview struct {
	ProductID string `json:"product_id"`
	SKUID     string `json:"sku_id"`
	skuLabel
	Requested int64 `json:"requested"`
	Available int64 `json:"available"`
	Short     int64 `json:"short"`
}

func (a *ConvertQuoteAction) Prepare(ctx context.Context, args json.RawMessage) (json.RawMessage, json.RawMessage, error) {
	var in convertArgs
	if err := decode(args, &in); err != nil {
		return nil, nil, err
	}
	if in.QuoteID == "" {
		return nil, nil, fmt.Errorf("%w: quote_id is required", ErrInvalidArguments)
	}
	q, err := a.quotes.Execute(ctx, in.QuoteID)
	if err != nil {
		return nil, nil, fmt.Errorf("quote %s: %w -- find it with list_quotes", in.QuoteID, err)
	}
	if len(q.Lines) == 0 {
		return nil, nil, fmt.Errorf("quote %s: %w", quotes.Reference(q.Number), ErrEmptyQuote)
	}
	labels, err := a.skuLabels(ctx, q.Lines)
	if err != nil {
		return nil, nil, err
	}
	warehouses, err := a.warehouses.Execute(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load warehouses: %w", err)
	}
	if len(warehouses) == 0 {
		return nil, nil, ErrNoWarehouse
	}
	warehouseID := warehouses[0].ID
	check, err := a.stock.Execute(ctx, q.ID, warehouseID)
	if err != nil {
		return nil, nil, fmt.Errorf("check stock: %w", err)
	}

	preview := convertPreview{
		QuoteID: q.ID, QuoteRef: quotes.Reference(q.Number), Name: q.Name, LineCount: len(q.Lines),
		TotalMinorUnits: q.Total(), CurrencyCode: q.CurrencyCode, WarehouseID: warehouseID,
		Warehouses: make([]warehousePreview, len(warehouses)), Shortfall: []shortfallPreview{},
	}
	for _, l := range q.Lines {
		preview.Lines = append(preview.Lines, convertLinePreview{
			ProductID: l.ProductID, SKUID: l.SKUID, skuLabel: labels[l.SKUID], Quantity: l.Quantity,
			UnitPriceMinorUnits: l.UnitPriceMinorUnits, LineTotalMinorUnits: l.Quantity * l.UnitPriceMinorUnits,
		})
	}
	if q.Customer != nil {
		preview.Customer = &customerPreview{ID: q.Customer.ID, Name: q.Customer.Name, Email: q.Customer.Email}
	}
	for i, w := range warehouses {
		preview.Warehouses[i] = warehousePreview{ID: w.ID, Name: w.Name}
	}
	for _, l := range check.Lines {
		if l.Short > 0 {
			preview.Shortfall = append(preview.Shortfall, shortfallPreview{
				ProductID: l.ProductID, SKUID: l.SKUID, skuLabel: labels[l.SKUID], Requested: l.Quantity, Available: l.Available, Short: l.Short,
			})
		}
	}

	payloadJSON, err := encode(convertPayload{QuoteID: q.ID, WarehouseID: warehouseID})
	if err != nil {
		return nil, nil, err
	}
	previewJSON, err := encode(preview)
	if err != nil {
		return nil, nil, err
	}
	return payloadJSON, previewJSON, nil
}

// skuLabels names the quote's SKUs from the price list, by SKU id.
func (a *ConvertQuoteAction) skuLabels(ctx context.Context, lines []quotes.Line) (map[string]skuLabel, error) {
	seen := map[string]bool{}
	var ids []string
	for _, l := range lines {
		if !seen[l.ProductID] {
			seen[l.ProductID] = true
			ids = append(ids, l.ProductID)
		}
	}
	priced, err := a.prices.Execute(ctx, apppms.ListPriceListInput{Page: 1, PerPage: len(ids), ProductIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("load product names: %w", err)
	}
	out := make(map[string]skuLabel, len(priced.Rows))
	for _, r := range priced.Rows {
		out[r.SKUID] = skuLabel{ProductName: r.ProductName, VariantName: r.VariantName, SKUCode: r.SKUCode}
	}
	return out, nil
}

func (a *ConvertQuoteAction) Execute(ctx context.Context, payload json.RawMessage, choices appassistant.ConfirmChoices) (appassistant.ActionResult, error) {
	var p convertPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return appassistant.ActionResult{}, fmt.Errorf("convertQuoteAction: decode payload: %w", err)
	}
	warehouseID := p.WarehouseID
	if choices.WarehouseID != "" {
		warehouseID = choices.WarehouseID
	}
	res, err := a.convert.Execute(ctx, appquotes.ConvertQuoteInput{QuoteID: p.QuoteID, WarehouseID: warehouseID})
	if err != nil {
		return appassistant.ActionResult{}, fmt.Errorf("convertQuoteAction: %w", err)
	}
	return appassistant.ActionResult{ID: res.SalesOrderID, Ref: res.Number}, nil
}
