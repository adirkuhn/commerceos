package woocommerce

import (
	"context"
	"fmt"
	"strconv"
	"time"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func (c *wooClient) ListOrders(ctx context.Context, page, perPage int, modifiedAfter *time.Time) ([]integrationsDomain.WooOrder, error) {
	// DatesAreGMT: true, paired with reading the *_gmt fields in
	// toWooOrder -- Woo's plain date_modified/date_created are the
	// store's local time (ambiguous without knowing its timezone
	// setting), the _gmt variants are always UTC. modified_after is
	// compared against whichever set DatesAreGMT selects.
	params := &wc.ListOrdersParams{Page: page, PerPage: perPage, OrderBy: "modified", Order: "asc", DatesAreGMT: true}
	if modifiedAfter != nil {
		params.ModifiedAfter = modifiedAfter.UTC().Format(time.RFC3339)
	}
	orders, _, err := c.client.Orders.List(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooOrder, len(orders))
	for i, o := range orders {
		out[i] = toWooOrder(o)
	}
	return out, nil
}

func (c *wooClient) GetOrder(ctx context.Context, externalOrderID string) (*integrationsDomain.WooOrder, error) {
	order, _, err := c.client.Orders.Get(ctx, externalOrderID, nil)
	if err != nil {
		return nil, err
	}
	result := toWooOrder(*order)
	return &result, nil
}

func (c *wooClient) UpdateOrderStatus(ctx context.Context, externalOrderID, status string) error {
	_, _, err := c.client.Orders.Update(ctx, externalOrderID, &wc.Order{Status: status})
	return err
}

func (c *wooClient) AddOrderNote(ctx context.Context, externalOrderID, note string) error {
	_, _, err := c.client.OrderNotes.Create(ctx, externalOrderID, &wc.OrderNote{Note: note, CustomerNote: false})
	return err
}

func (c *wooClient) ListCustomers(ctx context.Context, page, perPage int) ([]integrationsDomain.WooCustomer, error) {
	customers, _, err := c.client.Customers.List(ctx, &wc.ListCustomerParams{Page: page, PerPage: perPage})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooCustomer, len(customers))
	for i, cu := range customers {
		out[i] = toWooCustomer(cu)
	}
	return out, nil
}

// toWooOrderLineItemMetaData converts the underlying library's order-item
// meta_data (Value typed `any` -- Woo's REST API can carry non-string meta
// values for internal/plugin keys we don't care about) into this codebase's
// own WooMetaData shape, stringifying non-string values rather than
// dropping the entry -- the import use case only ever looks up one specific
// key (commerceos_expiry_date, always a plain date string), so a odd type
// on some unrelated key must never abort the whole conversion.
func toWooOrderLineItemMetaData(md []wc.MetaData) []integrationsDomain.WooMetaData {
	out := make([]integrationsDomain.WooMetaData, 0, len(md))
	for _, m := range md {
		value, ok := m.Value.(string)
		if !ok {
			value = fmt.Sprintf("%v", m.Value)
		}
		out = append(out, integrationsDomain.WooMetaData{Key: m.Key, Value: value})
	}
	return out
}

func toWooCustomer(cu wc.Customer) integrationsDomain.WooCustomer {
	var billing, shipping integrationsDomain.WooAddress
	if cu.Billing != nil {
		billing = integrationsDomain.WooAddress{
			FirstName: cu.Billing.FirstName, LastName: cu.Billing.LastName, Company: cu.Billing.Company,
			Address1: cu.Billing.Address1, Address2: cu.Billing.Address2, City: cu.Billing.City, State: cu.Billing.State,
			Postcode: cu.Billing.Postcode, Country: cu.Billing.Country, Email: cu.Billing.Email, Phone: cu.Billing.Phone,
		}
	}
	if cu.Shipping != nil {
		shipping = integrationsDomain.WooAddress{
			FirstName: cu.Shipping.FirstName, LastName: cu.Shipping.LastName, Company: cu.Shipping.Company,
			Address1: cu.Shipping.Address1, Address2: cu.Shipping.Address2, City: cu.Shipping.City, State: cu.Shipping.State,
			Postcode: cu.Shipping.Postcode, Country: cu.Shipping.Country, Phone: cu.Shipping.Phone,
		}
	}
	return integrationsDomain.WooCustomer{
		ID: cu.ID, Email: cu.Email, FirstName: cu.FirstName, LastName: cu.LastName,
		Username: cu.Username, Role: cu.Role, Billing: billing, Shipping: shipping,
		IsPayingCustomer: cu.IsPayingCustomer, AvatarURL: cu.AvatarURL,
		DateCreated: cu.DateCreatedGmt, DateModified: cu.DateModifiedGmt,
	}
}

func toWooAddress(a *wc.Billing) integrationsDomain.WooAddress {
	if a == nil {
		return integrationsDomain.WooAddress{}
	}
	return integrationsDomain.WooAddress{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Email: a.Email, Phone: a.Phone,
	}
}

func toWooShippingAddress(a *wc.Shipping) integrationsDomain.WooAddress {
	if a == nil {
		return integrationsDomain.WooAddress{}
	}
	return integrationsDomain.WooAddress{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Phone: a.Phone,
	}
}

func toWooOrder(o wc.Order) integrationsDomain.WooOrder {
	lineItems := make([]integrationsDomain.WooOrderLineItem, len(o.LineItems))
	for i, li := range o.LineItems {
		lineItems[i] = integrationsDomain.WooOrderLineItem{
			ProductID: li.ProductID, VariationID: li.VariationID, SKU: li.Sku, Name: li.Name, Quantity: int64(li.Quantity),
			Price:    strconv.FormatFloat(li.Price, 'f', -1, 64),
			Subtotal: li.Subtotal, SubtotalTax: li.SubtotalTax, Total: li.Total, TotalTax: li.TotalTax,
			MetaData: toWooOrderLineItemMetaData(li.MetaData),
		}
	}
	var shippingMethod, shippingMethodTitle string
	if len(o.ShippingLines) > 0 {
		shippingMethod = o.ShippingLines[0].MethodID
		shippingMethodTitle = o.ShippingLines[0].MethodTitle
	}
	return integrationsDomain.WooOrder{
		ID: o.ID, Number: o.Number, OrderKey: o.OrderKey, Status: o.Status, CustomerID: o.CustomerID,
		Billing: toWooAddress(o.Billing), Shipping: toWooShippingAddress(o.Shipping), CustomerNote: o.CustomerNote,
		Currency: o.Currency, DiscountTotal: o.DiscountTotal, ShippingTotal: o.ShippingTotal,
		TotalTax: o.TotalTax, Total: o.Total, PaymentMethod: o.PaymentMethod, PaymentMethodTitle: o.PaymentMethodTitle,
		ShippingMethod: shippingMethod, ShippingMethodTitle: shippingMethodTitle,
		TransactionID: o.TransactionID, DateCreated: o.DateCreatedGmt, DateModified: o.DateModifiedGmt,
		DatePaid: o.DatePaidGmt, DateCompleted: o.DateCompletedGmt, LineItems: lineItems,
	}
}
