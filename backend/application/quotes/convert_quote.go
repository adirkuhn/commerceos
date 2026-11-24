package quotes

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// quoteCatalog is pms' ListPriceListUseCase: each SKU's name, code and VAT
// rate, and whether the shop's prices include VAT.
type quoteCatalog interface {
	Execute(ctx context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error)
}

// orderCreator is oms' CreateSalesOrderUseCase. In a stock-reducing status
// it takes the stock (FEFO) and sends short lines to Stock issues.
type orderCreator interface {
	Execute(ctx context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error)
}

// shippingBookReader is customers' ListShippingAddressesUseCase.
type shippingBookReader interface {
	Execute(ctx context.Context, customerID string) (customersDomain.ShippingBook, error)
}

type ConvertQuoteDeps struct {
	Quotes    quotes.Repository
	Catalog   quoteCatalog
	Customers customerReader
	// ShippingAddresses reads the customer's book when the order ships to
	// a chosen one (ConvertQuoteInput.ShippingAddressID).
	ShippingAddresses shippingBookReader
	Orders            orderCreator
	Outbox            eventrecord.Enqueuer
	Now               func() time.Time
}

type ConvertQuoteInput struct {
	QuoteID     string
	WarehouseID string
	// Number is the order's own number; "" leaves it to the order's default.
	Number string
	// ShippingAddressID ships to one of the quote customer's shipping
	// addresses; "" ships to the customer's default one.
	ShippingAddressID string
}

type ConvertQuoteResult struct {
	SalesOrderID     string
	Number           string
	NeedsStockReview bool
}

// ConvertQuoteToSalesOrderUseCase turns a saved quote into a processing
// sales order at the quoted prices, taking the stock from one warehouse.
// Short lines don't stop it -- the order lands in Stock issues. A quote can
// be converted again (each time a new order). Each step runs in its own
// transaction, through the other modules' use cases: the quote read, the
// order (created by oms with its own stock movements), then the
// quotes.quote.converted event.
type ConvertQuoteToSalesOrderUseCase struct {
	db   *gorm.DB
	deps ConvertQuoteDeps
}

func NewConvertQuoteToSalesOrderUseCase(db *gorm.DB, deps ConvertQuoteDeps) *ConvertQuoteToSalesOrderUseCase {
	return &ConvertQuoteToSalesOrderUseCase{db: db, deps: deps}
}

func (uc *ConvertQuoteToSalesOrderUseCase) Execute(ctx context.Context, in ConvertQuoteInput) (*ConvertQuoteResult, error) {
	order, q, err := uc.buildOrder(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("convertQuoteToSalesOrder: %w", err)
	}
	created, err := uc.deps.Orders.Execute(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("convertQuoteToSalesOrder: %w", err)
	}
	result := &ConvertQuoteResult{
		SalesOrderID: created.SalesOrder.ID, Number: created.SalesOrder.Number, NeedsStockReview: created.SalesOrder.NeedsStockReview,
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return eventrecord.Record(ctx, uc.deps.Outbox, events.QuoteConverted, q.ID, eventrecord.Data{
			"sales_order_id": result.SalesOrderID, "number": q.Number, "needs_stock_review": result.NeedsStockReview,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("convertQuoteToSalesOrder: order %s created: %w", result.SalesOrderID, err)
	}
	return result, nil
}

func (uc *ConvertQuoteToSalesOrderUseCase) buildOrder(ctx context.Context, in ConvertQuoteInput) (appoms.CreateSalesOrderInput, *quotes.Quote, error) {
	q, err := loadQuote(ctx, uc.db, uc.deps.Quotes, in.QuoteID)
	if err != nil {
		return appoms.CreateSalesOrderInput{}, nil, err
	}
	var ordered []quotes.Line
	var productIDs []string
	for _, l := range q.Lines {
		if l.Quantity > 0 {
			ordered = append(ordered, l)
			productIDs = append(productIDs, l.ProductID)
		}
	}
	if len(ordered) == 0 {
		return appoms.CreateSalesOrderInput{}, nil, quotes.ErrNothingToOrder
	}
	catalog, err := uc.deps.Catalog.Execute(ctx, apppms.ListPriceListInput{ProductIDs: productIDs, PerPage: quotes.MaxLines})
	if err != nil {
		return appoms.CreateSalesOrderInput{}, nil, err
	}
	rowBySKU := make(map[string]apppms.PriceListRow, len(catalog.Rows))
	for _, r := range catalog.Rows {
		rowBySKU[r.SKUID] = r
	}

	order := appoms.CreateSalesOrderInput{
		WarehouseID: in.WarehouseID, Number: in.Number, QuoteID: &q.ID, CurrencyCode: q.CurrencyCode,
		Status: omsDomain.SalesOrderStatusProcessing, Source: omsDomain.SalesOrderSourceManual, OccurredAt: uc.deps.Now(),
	}
	amounts := make([]omsDomain.LineAmounts, 0, len(ordered))
	for _, l := range ordered {
		row, ok := rowBySKU[l.SKUID]
		if !ok {
			return appoms.CreateSalesOrderInput{}, nil, quotes.ErrLineUnavailable
		}
		a := omsDomain.PriceLine(l.Quantity, l.UnitPriceMinorUnits, row.TaxRateBasisPoints, catalog.PricesIncludeTax)
		amounts = append(amounts, a)
		order.LineItems = append(order.LineItems, appoms.CreateSalesOrderLineItemInput{
			SKUID: l.SKUID, Name: lineName(row), SKUCode: row.SKUCode, Quantity: l.Quantity, PriceMinorUnits: l.UnitPriceMinorUnits,
			SubtotalMinorUnits: a.Subtotal, SubtotalTaxMinorUnits: a.SubtotalTax, TotalMinorUnits: a.Total, TotalTaxMinorUnits: a.TotalTax,
			TotalIncludesTax: catalog.PricesIncludeTax,
		})
	}
	order.TotalMinorUnits, order.TaxTotalMinorUnits = omsDomain.OrderTotals(amounts, catalog.PricesIncludeTax)

	reference := quotes.Reference(q.Number)
	if q.Customer == nil {
		if in.ShippingAddressID != "" {
			return appoms.CreateSalesOrderInput{}, nil, customersDomain.ErrShippingAddressNotFound
		}
		// No customer to name the order: the quote's name says who it's for.
		order.CustomerNote = fmt.Sprintf("Quote %s — %s", reference, q.Name)
		return order, q, nil
	}
	customer, err := uc.deps.Customers.Execute(ctx, q.Customer.ID)
	if err != nil {
		return appoms.CreateSalesOrderInput{}, nil, err
	}
	order.CustomerID = &customer.ID
	shipping, err := uc.shipTo(ctx, customer, in.ShippingAddressID)
	if err != nil {
		return appoms.CreateSalesOrderInput{}, nil, err
	}
	order.Billing, order.Shipping = omsDomain.Address(customer.Billing), omsDomain.Address(shipping)
	order.CustomerNote = "Quote " + reference
	return order, q, nil
}

// shipTo is the chosen shipping address of the customer, or where their
// goods go by default (Customer.Shipping).
func (uc *ConvertQuoteToSalesOrderUseCase) shipTo(ctx context.Context, customer *customersDomain.Customer, addressID string) (customersDomain.Address, error) {
	if addressID == "" {
		return customer.Shipping, nil
	}
	book, err := uc.deps.ShippingAddresses.Execute(ctx, customer.ID)
	if err != nil {
		return customersDomain.Address{}, err
	}
	chosen, err := book.Find(addressID)
	if err != nil {
		return customersDomain.Address{}, err
	}
	return chosen.Address, nil
}

// lineName is the product name, plus the variant when it adds something.
func lineName(row apppms.PriceListRow) string {
	if row.VariantName != "" && row.VariantName != row.ProductName {
		return row.ProductName + " — " + row.VariantName
	}
	return row.ProductName
}
