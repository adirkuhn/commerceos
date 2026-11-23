package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/domain/assistant"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var orderStatuses = []omsDomain.SalesOrderStatus{
	omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold,
	omsDomain.SalesOrderStatusShipped, omsDomain.SalesOrderStatusCompleted, omsDomain.SalesOrderStatusCancelled,
	omsDomain.SalesOrderStatusRefunded, omsDomain.SalesOrderStatusPartiallyRefunded, omsDomain.SalesOrderStatusFailed,
}

var orderSources = []omsDomain.SalesOrderSource{
	omsDomain.SalesOrderSourcePOS, omsDomain.SalesOrderSourceManual, omsDomain.SalesOrderSourceWooCommerce,
	omsDomain.SalesOrderSourceBol, omsDomain.SalesOrderSourceWhatsApp, omsDomain.SalesOrderSourceInternal,
}

type salesOrderGetter interface {
	Execute(ctx context.Context, id string) (*appoms.GetSalesOrderResult, error)
}

type salesOrderLister interface {
	Execute(ctx context.Context, in appoms.ListSalesOrdersInput) (*appoms.ListSalesOrdersResult, error)
}

// GetSalesOrderTool reads one order by id or by its human order number.
type GetSalesOrderTool struct {
	get    salesOrderGetter
	search globalSearcher
}

func NewGetSalesOrderTool(get salesOrderGetter, search globalSearcher) *GetSalesOrderTool {
	return &GetSalesOrderTool{get: get, search: search}
}

func (t *GetSalesOrderTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_sales_order",
		Description: "Read one sales order -- status, customer, lines, totals, payment, shipping, fulfilment -- by its id " +
			"or its order number as the shop shows it (e.g. CO-12 or a WooCommerce/Bol number).",
		Parameters: object(map[string]any{"order": str("The order id (uuid) or order number.")}, "order"),
	}
}

type orderView struct {
	ID                      string          `json:"id"`
	Number                  string          `json:"number"`
	Status                  string          `json:"status"`
	Customer                orderCustomer   `json:"customer"`
	CustomerID              string          `json:"customer_id,omitempty"`
	QuoteID                 string          `json:"quote_id,omitempty"`
	ShippingAddress         string          `json:"shipping_address,omitempty"`
	CustomerNote            string          `json:"customer_note,omitempty"`
	Lines                   []orderLineView `json:"lines"`
	CurrencyCode            string          `json:"currency_code"`
	DiscountTotalMinorUnits int64           `json:"discount_total_minor_units"`
	ShippingTotalMinorUnits int64           `json:"shipping_total_minor_units"`
	TaxTotalMinorUnits      int64           `json:"tax_total_minor_units"`
	TotalMinorUnits         int64           `json:"total_minor_units"`
	CouponCode              string          `json:"coupon_code,omitempty"`
	PaymentMethod           string          `json:"payment_method,omitempty"`
	DatePaid                string          `json:"date_paid,omitempty"`
	DateCompleted           string          `json:"date_completed,omitempty"`
	ShippingMethod          string          `json:"shipping_method,omitempty"`
	ShipByDate              string          `json:"ship_by_date,omitempty"`
	Packed                  bool            `json:"packed"`
	Fulfilled               bool            `json:"fulfilled"`
	NeedsStockReview        bool            `json:"needs_stock_review"`
	NeedsSKUReview          bool            `json:"needs_sku_review"`
	OccurredAt              string          `json:"occurred_at"`
	UpdatedAt               string          `json:"updated_at"`
}

type orderCustomer struct {
	Name    string `json:"name"`
	Company string `json:"company,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

type orderLineView struct {
	Name                string `json:"name"`
	SKUCode             string `json:"sku_code"`
	Quantity            int64  `json:"quantity"`
	UnitPriceMinorUnits int64  `json:"unit_price_minor_units"`
	TotalMinorUnits     int64  `json:"total_minor_units"`
}

func (t *GetSalesOrderTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Order string `json:"order"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("order", args.Order); err != nil {
		return "", err
	}
	id := strings.TrimSpace(args.Order)
	if !uuidPattern.MatchString(id) {
		found, err := t.idForNumber(ctx, id)
		if err != nil {
			return "", err
		}
		id = found
	}
	res, err := t.get.Execute(ctx, id)
	if err != nil {
		return "", err
	}
	return toJSON(toOrderView(res))
}

// idForNumber resolves an order number through search, accepting only an
// exact (case-insensitive) number match -- search is fuzzy, and the model
// must never read a different order than the one asked for.
func (t *GetSalesOrderTool) idForNumber(ctx context.Context, number string) (string, error) {
	number = strings.TrimPrefix(strings.TrimSpace(number), "#")
	res, err := t.search.Execute(ctx, number)
	if err != nil {
		return "", err
	}
	var near []string
	for _, o := range res.SalesOrders {
		if strings.EqualFold(o.Number, number) {
			return o.ID, nil
		}
		near = append(near, o.Number)
	}
	if len(near) == 0 {
		return "", fmt.Errorf("no order numbered %q", number)
	}
	return "", fmt.Errorf("no order numbered %q; similar numbers: %s", number, strings.Join(near, ", "))
}

func toOrderView(res *appoms.GetSalesOrderResult) orderView {
	o := res.SalesOrder
	v := orderView{
		ID: o.ID, Number: o.Number, Status: string(o.Status),
		Customer: orderCustomer{
			Name: fullName(o.Billing.FirstName, o.Billing.LastName), Company: o.Billing.Company,
			Email: o.Billing.Email, Phone: o.Billing.Phone,
		},
		CustomerID: deref(o.CustomerID), QuoteID: deref(o.QuoteID),
		ShippingAddress: formatAddress(o.Shipping), CustomerNote: o.CustomerNote,
		Lines: make([]orderLineView, 0, len(res.LineItems)), CurrencyCode: o.CurrencyCode,
		DiscountTotalMinorUnits: o.DiscountTotalMinorUnits, ShippingTotalMinorUnits: o.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: o.TaxTotalMinorUnits, TotalMinorUnits: o.TotalMinorUnits, CouponCode: o.CouponCode,
		PaymentMethod: firstNonEmpty(o.PaymentMethodTitle, o.PaymentMethod),
		DatePaid:      formatTimePtr(o.DatePaid), DateCompleted: formatTimePtr(o.DateCompleted),
		ShippingMethod: firstNonEmpty(o.ShippingMethodTitle, o.ShippingMethod), ShipByDate: formatTimePtr(o.ShipByDate),
		Packed: o.Packed, Fulfilled: o.Fulfilled, NeedsStockReview: o.NeedsStockReview, NeedsSKUReview: o.NeedsSKUReview,
		OccurredAt: formatTime(o.OccurredAt), UpdatedAt: formatTime(o.UpdatedAt),
	}
	for _, l := range res.LineItems {
		v.Lines = append(v.Lines, orderLineView{
			Name: l.Name, SKUCode: l.SKUCode, Quantity: l.Quantity,
			UnitPriceMinorUnits: l.PriceMinorUnits, TotalMinorUnits: l.TotalMinorUnits,
		})
	}
	return v
}

func formatAddress(a omsDomain.Address) string {
	parts := []string{fullName(a.FirstName, a.LastName), a.Company, strings.TrimSpace(a.Address1 + " " + a.Address2),
		strings.TrimSpace(a.Postcode + " " + a.City), a.Country}
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

// ListSalesOrdersTool lists recent orders, optionally filtered.
type ListSalesOrdersTool struct{ list salesOrderLister }

func NewListSalesOrdersTool(list salesOrderLister) *ListSalesOrdersTool {
	return &ListSalesOrdersTool{list: list}
}

func (t *ListSalesOrdersTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_sales_orders",
		Description: "List sales orders, newest first, with the total count -- optionally only one status, one customer " +
			"(customer_id from search) or one source. Use get_sales_order for an order's lines.",
		Parameters: object(map[string]any{
			"status":      enum("Only orders in this status.", stringsOf(orderStatuses)...),
			"customer_id": str("Only this customer's orders (id from search)."),
			"source":      enum("Only orders from this source.", stringsOf(orderSources)...),
			"limit":       limitProp(),
		}),
	}
}

func (t *ListSalesOrdersTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Status     string `json:"status"`
		CustomerID string `json:"customer_id"`
		Source     string `json:"source"`
		Limit      int    `json:"limit"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	in := appoms.ListSalesOrdersInput{
		Page: 1, PerPage: clampLimit(args.Limit), SortBy: omsDomain.SalesOrderSortOccurredAt, SortDir: omsDomain.SortDesc,
	}
	if args.Status != "" {
		s := omsDomain.SalesOrderStatus(args.Status)
		if !contains(orderStatuses, s) {
			return "", fmt.Errorf("%w: unknown status %q", ErrInvalidArguments, args.Status)
		}
		in.Status = []omsDomain.SalesOrderStatus{s}
	}
	if args.Source != "" {
		s := omsDomain.SalesOrderSource(args.Source)
		if !contains(orderSources, s) {
			return "", fmt.Errorf("%w: unknown source %q", ErrInvalidArguments, args.Source)
		}
		in.Source = []omsDomain.SalesOrderSource{s}
	}
	if args.CustomerID != "" {
		in.CustomerID = &args.CustomerID
	}
	res, err := t.list.Execute(ctx, in)
	if err != nil {
		return "", err
	}
	out := struct {
		Total       int64      `json:"total"`
		SalesOrders []orderHit `json:"sales_orders"`
	}{Total: res.Total, SalesOrders: make([]orderHit, 0, len(res.SalesOrders))}
	for _, o := range res.SalesOrders {
		out.SalesOrders = append(out.SalesOrders, toOrderHit(o))
	}
	return toJSON(out)
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func contains[T comparable](values []T, v T) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
