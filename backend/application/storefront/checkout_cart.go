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
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
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
	// PaymentIntentID is set only by an embedded Stripe Elements checkout
	// (see CreatePaymentIntentUseCase) -- Execute re-verifies it server-side
	// (CardPaymentClient.RetrievePaymentIntent) against the freshly
	// computed order total before ever trusting it, same "never trust a
	// client-supplied price" rule ShippingMethodTitle's own doc comment
	// follows. Empty for the existing hosted-checkout-link flow, where
	// payment confirmation arrives later via webhook instead.
	PaymentIntentID string
}

type CheckoutResult struct {
	OrderNumber     string
	TotalMinorUnits int64
	CurrencyCode    string
	// PaymentURL is nil when the channel has no payment method configured,
	// or when the order was already paid via an embedded card payment
	// (CheckoutInput.PaymentIntentID) -- no redirect link is needed then.
	PaymentURL *string
}

// CheckoutEstimate is the same total EstimateTotal computes and Execute
// will (re)compute for real at checkout time -- used by
// CreatePaymentIntentUseCase to know the exact amount to charge before the
// order exists, without duplicating any of the subtotal/tax/shipping/
// discount logic.
type CheckoutEstimate struct {
	SubtotalMinorUnits int64
	TaxTotalMinorUnits int64
	ShippingMinorUnits int64
	DiscountMinorUnits int64
	TotalMinorUnits    int64
	CurrencyCode       string
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

// cardPaymentVerifier is *VerifyCardPaymentUseCase narrowed to point of use
// (ISP) -- may be nil (a channel/deployment that hasn't wired embedded
// card payment up), same optional-dependency convention as
// paymentLinkGenerator/deliveryPriceEstimator.
type cardPaymentVerifier interface {
	Execute(ctx context.Context, salesChannelID, paymentIntentID string) (*integrationsDomain.CardPaymentStatus, error)
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
	couponRepo          couponsDomain.Repository
	verifyCardPayment   cardPaymentVerifier
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
	couponRepo couponsDomain.Repository,
	verifyCardPayment cardPaymentVerifier,
) *CheckoutCartUseCase {
	return &CheckoutCartUseCase{
		db: db, cartRepo: cartRepo, customerRepo: customerRepo, createCustomer: createCustomer,
		createSalesOrder: createSalesOrder, taxClassRepo: taxClassRepo, taxRateRepo: taxRateRepo,
		outboxRepo: outboxRepo, generatePaymentLink: generatePaymentLink,
		shippingResolver:  &shippingOptionsResolver{db: db, connRepo: connRepo, shippingSettingsRepo: shippingSettingsRepo, estimateUC: estimateShippingUC},
		couponRepo:        couponRepo,
		verifyCardPayment: verifyCardPayment,
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

	totals, err := uc.computeTotals(ctx, cart, in.Shipping, in.ShippingMethodTitle)
	if err != nil {
		return nil, fmt.Errorf("checkoutCart: %w", err)
	}

	// An embedded Stripe Elements payment is never trusted client-side --
	// re-fetch the PaymentIntent from Stripe itself and only proceed when
	// it actually succeeded for exactly this order's total. A mismatch
	// (stale intent from a cart that changed since, wrong currency, a
	// forged id) fails checkout outright, same "never silently drop
	// something the shopper explicitly paid for/expects" stance
	// resolveCouponDiscount already takes for coupons.
	status := omsDomain.SalesOrderStatusPending
	var paymentProvider, paymentExternalID string
	var datePaid *time.Time
	if in.PaymentIntentID != "" {
		if uc.verifyCardPayment == nil {
			return nil, fmt.Errorf("checkoutCart: %w", storefrontDomain.ErrCardPaymentNotSupported)
		}
		cardStatus, err := uc.verifyCardPayment.Execute(ctx, cart.SalesChannelID, in.PaymentIntentID)
		if err != nil {
			return nil, fmt.Errorf("checkoutCart: verify card payment: %w", err)
		}
		if cardStatus.Status != "succeeded" || cardStatus.AmountMinorUnits != totals.TotalMinorUnits || !strings.EqualFold(cardStatus.CurrencyCode, totals.CurrencyCode) {
			return nil, fmt.Errorf("checkoutCart: %w", storefrontDomain.ErrCardPaymentMismatch)
		}
		status = omsDomain.SalesOrderStatusProcessing
		paymentProvider = "stripe"
		paymentExternalID = cardStatus.ID
		now := time.Now()
		datePaid = &now
	}

	orderNumber := "WEB-" + strings.ToUpper(idgen.NewV4()[:8])
	result, err := uc.createSalesOrder.Execute(ctx, appoms.CreateSalesOrderInput{
		Source: omsDomain.SalesOrderSourceInternal, Number: orderNumber, OrderKey: orderNumber,
		Status: status, CustomerID: &customerID, SalesChannelID: &cart.SalesChannelID,
		Billing:                 toOMSAddress(in.Billing, in.Email),
		Shipping:                toOMSAddress(in.Shipping, ""),
		CurrencyCode:            totals.CurrencyCode,
		TaxTotalMinorUnits:      totals.TaxTotalMinorUnits,
		ShippingTotalMinorUnits: totals.ShippingMinorUnits,
		ShippingMethod:          shippingMethodSlug(totals.ShippingMinorUnits),
		ShippingMethodTitle:     totals.ShippingMethodTitle,
		DiscountTotalMinorUnits: totals.DiscountMinorUnits,
		CouponCode:              totals.CouponCode,
		TotalMinorUnits:         totals.TotalMinorUnits,
		PaymentProvider:         paymentProvider,
		PaymentExternalID:       paymentExternalID,
		DatePaid:                datePaid,
		OccurredAt:              time.Now(),
		LineItems:               totals.LineItems,
	})
	if err != nil {
		return nil, fmt.Errorf("checkoutCart: create sales order: %w", err)
	}

	if err := uc.markCheckedOutAndPublish(ctx, cart, result.SalesOrder.ID); err != nil {
		return nil, fmt.Errorf("checkoutCart: %w", err)
	}

	// Best-effort, same non-atomicity tradeoff as markCheckedOutAndPublish
	// above: the order already exists, so a failure to bump the usage
	// counter must never fail checkout after the fact.
	if totals.CouponID != "" {
		_ = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			return uc.couponRepo.IncrementTimesUsed(ctx, totals.CouponID)
		})
	}

	checkoutResult := &CheckoutResult{OrderNumber: result.SalesOrder.Number, TotalMinorUnits: totals.TotalMinorUnits, CurrencyCode: totals.CurrencyCode}

	// Payment link generation is best-effort, same reasoning
	// application/whatsapp's placeOrder documents: it must never propagate
	// an error here (the order already exists and the cart is already
	// marked checked out) -- no payment method configured is a normal,
	// expected state, not a failure. Skipped entirely when the order was
	// already paid via an embedded card payment above -- no redirect link
	// is needed for an order that's already Processing.
	if uc.generatePaymentLink != nil && paymentExternalID == "" {
		link, ok, linkErr := uc.generatePaymentLink.Execute(ctx, cart.SalesChannelID, totals.TotalMinorUnits, totals.CurrencyCode, "Order #"+result.SalesOrder.Number, result.SalesOrder.Number)
		if linkErr == nil && ok {
			checkoutResult.PaymentURL = &link.URL
		}
	}
	return checkoutResult, nil
}

// EstimateTotal computes exactly what Execute would charge for token's
// cart right now, without creating an order -- used by
// CreatePaymentIntentUseCase to know the amount to start a Stripe
// PaymentIntent for before the order exists. Shares computeTotals with
// Execute so the two can never drift apart.
func (uc *CheckoutCartUseCase) EstimateTotal(ctx context.Context, token string, shipping CheckoutAddressInput, shippingMethodTitle string) (*CheckoutEstimate, error) {
	cart, err := uc.loadOpenCart(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("estimateCheckoutTotal: %w", err)
	}
	totals, err := uc.computeTotals(ctx, cart, shipping, shippingMethodTitle)
	if err != nil {
		return nil, fmt.Errorf("estimateCheckoutTotal: %w", err)
	}
	return &CheckoutEstimate{
		SubtotalMinorUnits: totals.SubtotalMinorUnits, TaxTotalMinorUnits: totals.TaxTotalMinorUnits,
		ShippingMinorUnits: totals.ShippingMinorUnits, DiscountMinorUnits: totals.DiscountMinorUnits,
		TotalMinorUnits: totals.TotalMinorUnits, CurrencyCode: totals.CurrencyCode,
	}, nil
}

// checkoutTotals is computeTotals' full result -- carries everything both
// Execute (to build the order) and EstimateTotal (to quote a
// PaymentIntent) need, computed exactly once per call so the two can never
// disagree on what a cart is about to cost.
type checkoutTotals struct {
	LineItems           []appoms.CreateSalesOrderLineItemInput
	SubtotalMinorUnits  int64
	TaxTotalMinorUnits  int64
	ShippingMinorUnits  int64
	ShippingMethodTitle string
	DiscountMinorUnits  int64
	CouponCode          string
	CouponID            string
	TotalMinorUnits     int64
	CurrencyCode        string
}

func (uc *CheckoutCartUseCase) computeTotals(ctx context.Context, cart *storefrontDomain.Cart, shipping CheckoutAddressInput, wantShippingTitle string) (*checkoutTotals, error) {
	classes, rates, err := uc.loadTaxConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load tax config: %w", err)
	}

	lineItems := make([]appoms.CreateSalesOrderLineItemInput, len(cart.Items))
	var subtotal, taxTotal int64
	currency := ""
	for i, item := range cart.Items {
		lineSubtotal := item.Quantity * item.PriceMinorUnits
		lineTax := computeLineTax(lineSubtotal, item.TaxClassID, shipping.Country, classes, rates)
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
	// out of, a failed MyParcel call) falls back to 0, never blocking
	// checkout.
	shippingAmount, shippingMethodTitle := uc.resolveShipping(ctx, shipping, wantShippingTitle)

	// Unlike shipping, a coupon the shopper explicitly attached is never
	// best-effort: if it's no longer redeemable (or the cart no longer
	// meets its minimum -- e.g. an item was removed after applying it),
	// checkout fails with a clear error rather than silently dropping the
	// discount the shopper is expecting.
	discountAmount, couponID, err := uc.resolveCouponDiscount(ctx, cart.CouponCode, subtotal)
	if err != nil {
		return nil, err
	}
	couponCode := ""
	if cart.CouponCode != nil {
		couponCode = *cart.CouponCode
	}

	total := subtotal + taxTotal + shippingAmount - discountAmount

	return &checkoutTotals{
		LineItems: lineItems, SubtotalMinorUnits: subtotal, TaxTotalMinorUnits: taxTotal,
		ShippingMinorUnits: shippingAmount, ShippingMethodTitle: shippingMethodTitle,
		DiscountMinorUnits: discountAmount, CouponCode: couponCode, CouponID: couponID,
		TotalMinorUnits: total, CurrencyCode: currency,
	}, nil
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

// resolveCouponDiscount returns (0, "", nil) when couponCode is nil (no
// coupon attached to this cart). Otherwise re-loads the coupon fresh (see
// Cart.CouponCode's own doc comment on why the cart never caches a
// discount amount) and re-checks both IsRedeemable and MeetsMinSubtotal --
// either failing returns an error that Execute propagates, never silently
// drops the discount. Returns the coupon's ID (for IncrementTimesUsed)
// alongside the amount.
func (uc *CheckoutCartUseCase) resolveCouponDiscount(ctx context.Context, couponCode *string, subtotal int64) (int64, string, error) {
	if couponCode == nil {
		return 0, "", nil
	}
	var coupon *couponsDomain.Coupon
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		coupon, err = uc.couponRepo.FindByCode(ctx, *couponCode)
		return err
	})
	if err != nil {
		return 0, "", err
	}
	if coupon == nil || !coupon.IsRedeemable(time.Now()) {
		return 0, "", couponsDomain.ErrCouponNotRedeemable
	}
	if !coupon.MeetsMinSubtotal(subtotal) {
		return 0, "", couponsDomain.ErrCouponMinSubtotalNotMet
	}
	return coupon.DiscountForSubtotal(subtotal), coupon.ID, nil
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
