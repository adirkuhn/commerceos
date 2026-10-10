package storefront

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	"github.com/adirkuhn/commerceos/application/shopperemail"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
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
	channelRepo         channelsDomain.SalesChannelRepository
	customerRepo        customersDomain.CustomerRepository
	createCustomer      customerCreator
	createSalesOrder    salesOrderCreator
	taxClassRepo        taxDomain.TaxClassRepository
	taxRateRepo         taxDomain.TaxRateRepository
	outboxRepo          outbox.Repository
	generatePaymentLink paymentLinkGenerator
	shippingResolver    *shippingOptionsResolver
	couponRepo          couponsDomain.Repository
	couponCurrencyRepo  couponsDomain.CouponCurrencyAmountRepository
	fxRepo              fxDomain.ExchangeRateRepository
	verifyCardPayment   cardPaymentVerifier
}

func NewCheckoutCartUseCase(
	db *gorm.DB,
	cartRepo storefrontDomain.CartRepository,
	channelRepo channelsDomain.SalesChannelRepository,
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
	couponCurrencyRepo couponsDomain.CouponCurrencyAmountRepository,
	fxRepo fxDomain.ExchangeRateRepository,
	verifyCardPayment cardPaymentVerifier,
) *CheckoutCartUseCase {
	return &CheckoutCartUseCase{
		db: db, cartRepo: cartRepo, channelRepo: channelRepo, customerRepo: customerRepo, createCustomer: createCustomer,
		createSalesOrder: createSalesOrder, taxClassRepo: taxClassRepo, taxRateRepo: taxRateRepo,
		outboxRepo: outboxRepo, generatePaymentLink: generatePaymentLink,
		shippingResolver:   &shippingOptionsResolver{db: db, connRepo: connRepo, shippingSettingsRepo: shippingSettingsRepo, estimateUC: estimateShippingUC},
		couponRepo:         couponRepo,
		couponCurrencyRepo: couponCurrencyRepo,
		fxRepo:             fxRepo,
		verifyCardPayment:  verifyCardPayment,
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

	channel, err := uc.resolveChannel(ctx, cart.SalesChannelID)
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
		WarehouseID:             channel.warehouseID,
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
		if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			return uc.couponRepo.IncrementTimesUsed(ctx, totals.CouponID)
		}); err != nil {
			log.Printf("checkoutCart: increment coupon %s usage (order already placed): %v", totals.CouponID, err)
		}
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
		link, ok, linkErr := uc.generatePaymentLink.Execute(ctx, cart.SalesChannelID, totals.TotalMinorUnits, totals.CurrencyCode, shopperemail.PaymentDescription(channel.language, result.SalesOrder.Number), result.SalesOrder.Number)
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
