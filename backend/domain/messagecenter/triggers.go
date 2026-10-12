package messagecenter

import "github.com/adirkuhn/commerceos/domain/notifications"

// VariableType says how a template uses a variable.
type VariableType string

const (
	// VariableText is printed as {{name}}.
	VariableText VariableType = "text"
	// VariableList is repeated with {{#name}}...{{/name}}; Fields are what
	// each repetition offers.
	VariableList VariableType = "list"
)

// Variable is one value a trigger's templates can use. Example is what
// previews show (and documents the shape).
type Variable struct {
	Name    string
	Type    VariableType
	Example any
	Fields  []Variable
}

// Trigger is a domain event an AutomationRule can fire on: who its messages
// go to, where its order is (OrderIDField, "" when it has none -- the
// payload field holding the sales order id, whose details the engine then
// loads), and every variable its templates can use.
type Trigger struct {
	Subject      string
	Audience     Audience
	OrderIDField string
	// EventVariables are the event's own payload fields; Variables adds the
	// store's, and the order's when there is one.
	EventVariables []Variable
}

// Variables lists everything this trigger's templates can use.
func (t Trigger) Variables() []Variable {
	out := append([]Variable{}, StoreVariables...)
	if t.OrderIDField != "" {
		out = append(out, OrderVariables...)
	}
	return append(out, t.EventVariables...)
}

// SampleData is example data for every variable -- what template previews
// render.
func (t Trigger) SampleData() map[string]any {
	data := map[string]any{}
	for _, v := range t.Variables() {
		data[v.Name] = sampleValue(v)
	}
	return data
}

func sampleValue(v Variable) any {
	if v.Type != VariableList {
		return v.Example
	}
	row := map[string]any{}
	for _, f := range v.Fields {
		row[f.Name] = f.Example
	}
	return []any{row, row}
}

// TriggerStockCloseToExpiry is the daily stock-close-to-expiry report (one
// event per report window) -- see domain/events.StockCloseToExpiryReported.
const TriggerStockCloseToExpiry = "inventory.stock.close_to_expiry_reported"

// AudienceKind is who a trigger's messages go to by default.
type AudienceKind string

const (
	// AudienceCustomer: the customer of the trigger's sales order (the
	// order's billing contact for a guest order); nobody when the trigger
	// has no order.
	AudienceCustomer AudienceKind = "customer"
	// AudienceStaff: the tenant's own team -- the notification email
	// recipients of Audience.StaffCategory, when that category's email
	// toggle is on.
	AudienceStaff AudienceKind = "staff"
)

// Audience is who a trigger's messages go to, before a rule's
// ExtraRecipients are added.
type Audience struct {
	Kind AudienceKind
	// StaffCategory is the notification category (see
	// domain/notifications.Category) whose email recipients an
	// AudienceStaff trigger goes to.
	StaffCategory string
}

func text(name string, example any) Variable {
	return Variable{Name: name, Type: VariableText, Example: example}
}

// StoreVariables are available to every trigger.
var StoreVariables = []Variable{
	text("store_name", "Cantinho Brasileiro"),
	text("store_email", "hello@example.com"),
	text("store_phone", "+31 20 123 4567"),
	text("store_address", "Damrak 1, 1012 LG Amsterdam, NL"),
}

// OrderVariables are available to every trigger with an order.
var OrderVariables = []Variable{
	text("customer_name", "Ana"),
	text("customer_first_name", "Ana"),
	text("customer_last_name", "Souza"),
	text("customer_email", "ana@example.com"),
	text("order_number", "1001"),
	text("order_date", "10/10/2026"),
	text("order_status", "processing"),
	text("order_source", "internal"),
	text("order_status_label", "Processing"),
	{Name: "items", Type: VariableList, Fields: []Variable{
		text("name", "Pão de queijo 400g"),
		text("sku", "PQ-400"),
		text("quantity", 2),
		text("unit_price", "€ 4,99"),
		text("total", "€ 9,98"),
	}},
	text("item_count", 2),
	text("subtotal", "€ 9,98"),
	text("shipping_total", "€ 4,95"),
	text("discount_total", "€ 0,00"),
	text("tax_total", "€ 1,73"),
	text("order_total", "€ 14,93"),
	text("shipping_name", "Ana Souza"),
	text("shipping_address", "Kalverstraat 10, 1012 PA Amsterdam, NL"),
	text("shipping_method", "standard"),
	text("payment_method", "iDEAL"),
	text("customer_note", ""),
	text("carrier", "PostNL"),
	text("tracking_barcode", "3SABCD1234567"),
	text("tracking_url", "https://jouw.postnl.nl/track-and-trace/3SABCD1234567"),
}

// Triggers is every domain event an AutomationRule can fire on.
var Triggers = []Trigger{
	{Subject: "storefront.order.placed", Audience: Audience{Kind: AudienceCustomer}, OrderIDField: "sales_order_id"},
	{Subject: "oms.sales_order.created", Audience: Audience{Kind: AudienceCustomer}, OrderIDField: "entity_id",
		EventVariables: []Variable{text("source", "woocommerce")}},
	{Subject: "oms.sales_order.status_changed", Audience: Audience{Kind: AudienceCustomer}, OrderIDField: "sales_order_id"},
	{Subject: "shipping.shipment.fulfillment_signal", Audience: Audience{Kind: AudienceCustomer}, OrderIDField: "sales_order_id",
		EventVariables: []Variable{text("kind", "shipment_status"), text("status_code", 8), text("status_main", "delivered")}},
	{Subject: "payments.refund.issued", Audience: Audience{Kind: AudienceCustomer}, OrderIDField: "sales_order_id",
		EventVariables: []Variable{text("quantity", 1), text("refund_amount", "€ 4,99")}},
	{Subject: "payments.refund.manual_review_required", Audience: Audience{Kind: AudienceCustomer}, OrderIDField: "sales_order_id",
		EventVariables: []Variable{text("quantity", 1), text("refund_amount", "€ 4,99")}},
	{Subject: TriggerStockCloseToExpiry, Audience: Audience{Kind: AudienceStaff, StaffCategory: string(notifications.CategoryStock)},
		EventVariables: []Variable{
			text("window_days", 30), text("min_days_left", 8), text("item_count", 2),
			{Name: "items", Type: VariableList, Fields: []Variable{
				text("product_name", "Pão de queijo 400g"), text("sku", "PQ-400"),
				text("image_url", "https://placehold.co/112x112/png"), text("quantity", 7),
				text("expiry_date", "30/10/2026"), text("days_left", 19), text("price", "€ 4,99"), text("sale_price", "€ 3,99"),
			}},
		}},
	{Subject: "pms.product.changed", Audience: Audience{Kind: AudienceCustomer}, EventVariables: []Variable{text("product_id", "…")}},
	{Subject: "pms.product.deleted", Audience: Audience{Kind: AudienceCustomer}, EventVariables: []Variable{text("product_id", "…")}},
	{Subject: "pms.category.changed", Audience: Audience{Kind: AudienceCustomer}, EventVariables: []Variable{text("category_id", "…")}},
	{Subject: "integrations.connection.connected", Audience: Audience{Kind: AudienceCustomer},
		EventVariables: []Variable{text("provider", "woocommerce"), text("connection_id", "…")}},
	{Subject: "inventory.stock.changed", Audience: Audience{Kind: AudienceCustomer}, EventVariables: []Variable{text("sku_id", "…")}},
}

// TriggerFor finds subject's trigger.
func TriggerFor(subject string) (Trigger, bool) {
	for _, t := range Triggers {
		if t.Subject == subject {
			return t, true
		}
	}
	return Trigger{}, false
}

// KnownTriggerSubjects lists every trigger's subject.
func KnownTriggerSubjects() []string {
	out := make([]string, len(Triggers))
	for i, t := range Triggers {
		out[i] = t.Subject
	}
	return out
}

// ValidTriggerSubject reports whether a rule may fire on subject.
func ValidTriggerSubject(subject string) bool {
	_, ok := TriggerFor(subject)
	return ok
}

// AudienceFor returns who subject's messages go to by default.
func AudienceFor(subject string) Audience {
	if t, ok := TriggerFor(subject); ok {
		return t.Audience
	}
	return Audience{Kind: AudienceCustomer}
}
