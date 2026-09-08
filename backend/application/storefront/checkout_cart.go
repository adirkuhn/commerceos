package storefront

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// CheckoutAddressInput mirrors customers.Address's shape -- domain packages
// never import each other in this codebase, so oms/customers each already
// keep their own copy; this is the application-layer input shape that maps
// onto both.
type CheckoutAddressInput struct {
	FirstName, LastName, Company              string
	Address1, Address2, City, State, Postcode string
	Country                                   string
	Phone                                     string
}

type CheckoutInput struct {
	Email    string
	Billing  CheckoutAddressInput
	Shipping CheckoutAddressInput
	// ShippingMethodTitle is which of GetShippingOptionsUseCase's options
	// (by its Title -- see ShippingOption's doc comment on why that's the
	// identity) the shopper picked, or "" for "let the server pick" --
	// never a price: Execute always re-resolves the real options itself
	// and looks this title up among them, so a client can never inject an
	// arbitrary shipping cost. A title that no longer matches any current
	// option (the shopper's session outlived the quote) falls back to the
	// cheapest option, same as "" does.
	ShippingMethodTitle string
}

type CheckoutResult struct {
	OrderNumber     string
	TotalMinorUnits int64
	CurrencyCode    string
	// PaymentURL is nil when the channel has no payment method configured.
	PaymentURL *string
}

// salesOrderCreator is *application/oms.CreateSalesOrderUseCase narrowed to
// point of use (ISP), same convention as application/whatsapp's use of the
// same use case.
type salesOrderCreator interface {
	Execute(ctx context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error)
}

// customerCreator is *application/customers.CreateCustomerUseCase narrowed
// to point of use (ISP), same convention as application/whatsapp's use of
// the same use case.
type customerCreator interface {
	Execute(ctx context.Context, in appcustomers.CreateCustomerInput) (*customersDomain.Customer, error)
}

// paymentLinkGenerator is *GeneratePaymentLinkUseCase narrowed to point of
// use (ISP) -- may be nil (a deployment/channel that hasn't wired payment
// yet), same optional-dependency convention as
// application/whatsapp.paymentLinkGenerator.
type paymentLinkGenerator interface {
	Execute(ctx context.Context, salesChannelID string, amountMinorUnits int64, currencyCode, description, reference string) (*integrationsDomain.PaymentLink, bool, error)
}

// deliveryPriceEstimator is *application/shipping.GetDeliveryPriceEstimateUseCase
// narrowed to point of use (ISP) -- may be nil (deployments with no
// shipping estimate wired up yet), same optional-dependency convention as
// paymentLinkGenerator.
type deliveryPriceEstimator interface {
	Execute(ctx context.Context, in appshipping.GetDeliveryPriceEstimateInput) ([]shippingDomain.DeliveryPriceOption, error)
}

type CheckoutCartUseCase struct {
	db                  *gorm.DB
	cartRepo            storefrontDomain.CartRepository
	customerRepo        customersDomain.CustomerRepository
	createCustomer      customerCreator
	createSalesOrder    salesOrderCreator
	taxClassRepo        taxDomain.TaxClassRepository
	taxRateRepo         taxDomain.TaxRateRepository
	outboxRepo          outbox.Repository
	generatePaymentLink paymentLinkGenerator
	shippingResolver    *shippingOptionsResolver
}

func NewCheckoutCartUseCase(
	db *gorm.DB,
	cartRepo storefrontDomain.CartRepository,
	customerRepo customersDomain.CustomerRepository,
	createCustomer customerCreator,
	createSalesOrder salesOrderCreator,
	taxClassRepo taxDomain.TaxClassRepository,
	taxRateRepo taxDomain.TaxRateRepository,
	outboxRepo outbox.Repository,
	generatePaymentLink paymentLinkGenerator,
	connRepo integrationsDomain.ConnectionRepository,
	shippingSettingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	estimateShippingUC deliveryPriceEstimator,
) *CheckoutCartUseCase {
	return &CheckoutCartUseCase{
		db: db, cartRepo: cartRepo, customerRepo: customerRepo, createCustomer: createCustomer,
		createSalesOrder: createSalesOrder, taxClassRepo: taxClassRepo, taxRateRepo: taxRateRepo,
		outboxRepo: outboxRepo, generatePaymentLink: generatePaymentLink,
		shippingResolver: &shippingOptionsResolver{db: db, connRepo: connRepo, shippingSettingsRepo: shippingSettingsRepo, estimateUC: estimateShippingUC},
	}
}

// Execute turns token's cart into a Pending oms.SalesOrder. Guest checkout
// only -- finds or creates a Customer by email, same convention
// WooCommerce guest-checkout import already uses
// (customers.CustomerRepository.FindByEmail's doc comment). Stock is not
// re-reserved here (it was checked at UpdateCartItemsUseCase time) and the
// order is never anything but Pending -- same oversell-window tradeoff
// application/whatsapp's chat checkout already accepts, not a new risk.
func (uc *CheckoutCartUseCase) Execute(ctx context.Context, token string, in CheckoutInput) (*CheckoutResult, error) {
	cart, err := uc.loadOpenCart(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("checkoutCart: %w", err)
	}

	customerID, err := uc.resolveCustomer(ctx, in.Email, in.Billing, in.Shipping)
	if err != nil {
		return nil, fmt.Errorf("checkoutCart: resolve customer: %w", err)
	}

	classes, rates, err := uc.loadTaxConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("checkoutCart: load tax config: %w", err)
	}

	lineItems := make([]appoms.CreateSalesOrderLineItemInput, len(cart.Items))
	var subtotal, taxTotal int64
	currency := ""
	for i, item := range cart.Items {
		lineSubtotal := item.Quantity * item.PriceMinorUnits
		lineTax := computeLineTax(lineSubtotal, item.TaxClassID, in.Shipping.Country, classes, rates)
		lineItems[i] = appoms.CreateSalesOrderLineItemInput{
			SKUID: item.SKUID, Name: item.ProductName, SKUCode: item.SKUCode, Quantity: item.Quantity,
			PriceMinorUnits: item.PriceMinorUnits, SubtotalMinorUnits: lineSubtotal, SubtotalTaxMinorUnits: lineTax,
			TotalMinorUnits: lineSubtotal, TotalTaxMinorUnits: lineTax,
		}
		subtotal += lineSubtotal
		taxTotal += lineTax
		currency = item.CurrencyCode
	}
	// Best-effort: NL/BE only (MyParcel's own API constraint), and only
	// when a MyParcel connection + a house number are both available --
	// see estimateShipping's own doc comment. Every other case (no
	// connection, unsupported country, address we can't parse a number
	// out of, a failed MyParcel call) falls back to 0, same as before this
	// existed, never blocking checkout.
	shippingAmount, shippingMethodTitle := uc.resolveShipping(ctx, in.Shipping, in.ShippingMethodTitle)
	total := subtotal + taxTotal + shippingAmount

	orderNumber := "WEB-" + strings.ToUpper(idgen.NewV4()[:8])
	result, err := uc.createSalesOrder.Execute(ctx, appoms.CreateSalesOrderInput{
		Source: omsDomain.SalesOrderSourceInternal, Number: orderNumber, OrderKey: orderNumber,
		Status: omsDomain.SalesOrderStatusPending, CustomerID: &customerID, SalesChannelID: &cart.SalesChannelID,
		Billing:                 toOMSAddress(in.Billing, in.Email),
		Shipping:                toOMSAddress(in.Shipping, ""),
		CurrencyCode:            currency,
		TaxTotalMinorUnits:      taxTotal,
		ShippingTotalMinorUnits: shippingAmount,
		ShippingMethod:          shippingMethodSlug(shippingAmount),
		ShippingMethodTitle:     shippingMethodTitle,
		TotalMinorUnits:         total,
		OccurredAt:              time.Now(),
		LineItems:               lineItems,
	})
	if err != nil {
		return nil, fmt.Errorf("checkoutCart: create sales order: %w", err)
	}

	if err := uc.markCheckedOutAndPublish(ctx, cart, result.SalesOrder.ID); err != nil {
		return nil, fmt.Errorf("checkoutCart: %w", err)
	}

	checkoutResult := &CheckoutResult{OrderNumber: result.SalesOrder.Number, TotalMinorUnits: total, CurrencyCode: currency}

	// Payment link generation is best-effort, same reasoning
	// application/whatsapp's placeOrder documents: it must never propagate
	// an error here (the order already exists and the cart is already
	// marked checked out) -- no payment method configured is a normal,
	// expected state, not a failure.
	if uc.generatePaymentLink != nil {
		link, ok, linkErr := uc.generatePaymentLink.Execute(ctx, cart.SalesChannelID, total, currency, "Order #"+result.SalesOrder.Number, result.SalesOrder.Number)
		if linkErr == nil && ok {
			checkoutResult.PaymentURL = &link.URL
		}
	}
	return checkoutResult, nil
}

func (uc *CheckoutCartUseCase) loadOpenCart(ctx context.Context, token string) (*storefrontDomain.Cart, error) {
	var cart *storefrontDomain.Cart
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.cartRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrCartNotFound
		}
		if found.IsCheckedOut() {
			return storefrontDomain.ErrCartAlreadyCheckedOut
		}
		if len(found.Items) == 0 {
			return storefrontDomain.ErrCartEmpty
		}
		cart = found
		return nil
	})
	return cart, err
}

func (uc *CheckoutCartUseCase) resolveCustomer(ctx context.Context, email string, billing, shipping CheckoutAddressInput) (string, error) {
	var existing *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.customerRepo.FindByEmail(ctx, email)
		if err != nil {
			return err
		}
		existing = found
		return nil
	})
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.ID, nil
	}

	created, err := uc.createCustomer.Execute(ctx, appcustomers.CreateCustomerInput{
		Email: email, FirstName: billing.FirstName, LastName: billing.LastName,
		Billing:  toCustomerAddress(billing, email),
		Shipping: toCustomerAddress(shipping, ""),
	})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

// resolveShipping re-resolves the real shipping options for shipping
// itself (never trusts a client-supplied price -- see CheckoutInput's doc
// comment on ShippingMethodTitle) and picks wantTitle among them if it's
// non-empty and still matches one, else the cheapest. No options
// available at all (see shippingOptionsResolver's doc comment for every
// reason that happens) falls back to (0, ""), never blocking checkout.
func (uc *CheckoutCartUseCase) resolveShipping(ctx context.Context, shipping CheckoutAddressInput, wantTitle string) (int64, string) {
	options := uc.shippingResolver.resolve(ctx, shipping)
	if len(options) == 0 {
		return 0, ""
	}
	if wantTitle != "" {
		for _, o := range options {
			if o.Title == wantTitle {
				return o.AmountMinorUnits, o.Title
			}
		}
	}
	cheapest := options[0]
	for _, o := range options[1:] {
		if o.AmountMinorUnits < cheapest.AmountMinorUnits {
			cheapest = o
		}
	}
	return cheapest.AmountMinorUnits, cheapest.Title
}

// shippingMethodSlug is the machine-readable counterpart to the human
// title estimateShipping returns -- empty when no estimate was applied,
// same "both empty means no shipping method was recorded" convention
// oms.SalesOrder.ShippingMethod's own doc comment describes.
func shippingMethodSlug(shippingAmount int64) string {
	if shippingAmount == 0 {
		return ""
	}
	return "myparcel"
}

func (uc *CheckoutCartUseCase) loadTaxConfig(ctx context.Context) ([]taxDomain.TaxClass, []taxDomain.TaxRate, error) {
	var classes []taxDomain.TaxClass
	var rates []taxDomain.TaxRate
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if classes, err = uc.taxClassRepo.List(ctx); err != nil {
			return err
		}
		rates, err = uc.taxRateRepo.List(ctx)
		return err
	})
	return classes, rates, err
}

// markCheckedOutAndPublish sets Cart.CheckedOutAt and enqueues
// StorefrontOrderPlaced in one transaction -- deliberately a separate
// transaction from the order+line-items write above (uc.createSalesOrder
// opens its own), same accepted non-atomicity tradeoff
// application/oms.CreateSalesOrderUseCase's own doc comment already
// documents for its stock-movement recording. A crash between the two
// leaves an order with no corresponding cart-closed flag/event -- rare,
// and not worse than what CreateSalesOrderUseCase already accepts.
func (uc *CheckoutCartUseCase) markCheckedOutAndPublish(ctx context.Context, cart *storefrontDomain.Cart, salesOrderID string) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		cart.CheckedOutAt = &now
		cart.UpdatedAt = now
		if err := uc.cartRepo.Update(ctx, cart); err != nil {
			return err
		}
		event := events.StorefrontOrderPlaced{
			Base:         events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID:     tenant.IDFromContext(ctx),
			SalesOrderID: salesOrderID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
}

// computeLineTax resolves the effective tax class (the line's own, else
// the tenant's default) and country rate, applying it to lineSubtotal --
// 0 when no matching rate is configured, same "unset means unset, never an
// error" convention as settings.TenantSettings' zero-value fields.
func computeLineTax(lineSubtotal int64, taxClassID *string, country string, classes []taxDomain.TaxClass, rates []taxDomain.TaxRate) int64 {
	classID := ""
	if taxClassID != nil {
		classID = *taxClassID
	} else {
		for _, c := range classes {
			if c.IsDefault {
				classID = c.ID
				break
			}
		}
	}
	if classID == "" {
		return 0
	}
	for _, r := range rates {
		if r.TaxClassID == classID && strings.EqualFold(r.Country, country) {
			return lineSubtotal * int64(r.RateBasisPoints) / 10000
		}
	}
	return 0
}

func toOMSAddress(in CheckoutAddressInput, email string) omsDomain.Address {
	return omsDomain.Address{
		FirstName: in.FirstName, LastName: in.LastName, Company: in.Company,
		Address1: in.Address1, Address2: in.Address2, City: in.City, State: in.State,
		Postcode: in.Postcode, Country: in.Country, Email: email, Phone: in.Phone,
	}
}

func toCustomerAddress(in CheckoutAddressInput, email string) customersDomain.Address {
	return customersDomain.Address{
		FirstName: in.FirstName, LastName: in.LastName, Company: in.Company,
		Address1: in.Address1, Address2: in.Address2, City: in.City, State: in.State,
		Postcode: in.Postcode, Country: in.Country, Email: email, Phone: in.Phone,
	}
}
