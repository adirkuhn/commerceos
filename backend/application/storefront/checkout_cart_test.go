package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

// fakeSalesOrderCreator captures the CreateSalesOrderInput it was called
// with -- checkout's own logic (tax math, line mapping) is what these
// tests actually verify.
type fakeSalesOrderCreator struct {
	got appoms.CreateSalesOrderInput
	err error
}

func (f *fakeSalesOrderCreator) Execute(_ context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error) {
	f.got = in
	if f.err != nil {
		return nil, f.err
	}
	return &appoms.CreateSalesOrderResult{SalesOrder: omsDomain.SalesOrder{ID: "order-1", Number: in.Number}}, nil
}

type fakeCustomerCreator struct {
	out *customersDomain.Customer
	err error
}

func (f *fakeCustomerCreator) Execute(_ context.Context, in appcustomers.CreateCustomerInput) (*customersDomain.Customer, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.out != nil {
		return f.out, nil
	}
	return &customersDomain.Customer{ID: "new-customer-1", Email: in.Email}, nil
}

type fakePaymentLinkGenerator struct {
	link *integrationsDomain.PaymentLink
	ok   bool
	err  error
}

// Execute is nil-receiver-safe -- passing a nil *fakePaymentLinkGenerator
// through the exported paymentLinkGenerator interface parameter (as tests
// exercising "channel has no payment method wired at all" do) would
// otherwise wrap a non-nil interface around a nil pointer, the classic
// typed-nil trap, rather than behave like a real "not configured" case.
func (f *fakePaymentLinkGenerator) Execute(_ context.Context, _ string, _ int64, _, _, _ string) (*integrationsDomain.PaymentLink, bool, error) {
	if f == nil {
		return nil, false, nil
	}
	return f.link, f.ok, f.err
}

func validCheckoutInput() appstorefront.CheckoutInput {
	addr := appstorefront.CheckoutAddressInput{FirstName: "A", LastName: "B", Address1: "Street 1", City: "Amsterdam", Postcode: "1000AA", Country: "NL"}
	return appstorefront.CheckoutInput{Email: "shopper@example.com", Billing: addr, Shipping: addr}
}

func newCheckoutUseCase(t *testing.T, n int, cartRepo *fakeCartRepo, customerRepo *fakeCustomerRepo, createCustomer *fakeCustomerCreator, createOrder *fakeSalesOrderCreator, taxClassRepo *fakeTaxClassRepo, taxRateRepo *fakeTaxRateRepo, outboxRepo *fakeOutboxRepo, paymentGen *fakePaymentLinkGenerator) *appstorefront.CheckoutCartUseCase {
	return appstorefront.NewCheckoutCartUseCase(newMockGormDB(t, n), cartRepo, customerRepo, createCustomer, createOrder, taxClassRepo, taxRateRepo, outboxRepo, paymentGen)
}

func openCartWithOneItem() storefrontDomain.Cart {
	return storefrontDomain.Cart{
		ID: "cart-1", SalesChannelID: "channel-1", Token: "token-1",
		Items: []storefrontDomain.CartItem{{SKUID: "sku-1", SKUCode: "SKU-A", ProductName: "Widget", Quantity: 2, PriceMinorUnits: 500, CurrencyCode: "EUR"}},
	}
}

// 4 transactions: loadOpenCart, resolveCustomer's FindByEmail read,
// loadTaxConfig, markCheckedOutAndPublish -- see CheckoutCartUseCase's doc
// comment on why customer-creation and order-creation (both fakes here)
// never count towards this.
const checkoutTxnCount = 4

func TestCheckoutCart_placesAPendingOrderForAnExistingCustomer(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	createOrder := &fakeSalesOrderCreator{}
	outboxRepo := &fakeOutboxRepo{}
	uc := newCheckoutUseCase(t, checkoutTxnCount, cartRepo, customerRepo, &fakeCustomerCreator{}, createOrder, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, outboxRepo, nil)

	result, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.NoError(t, err)
	require.Equal(t, "sku-1", createOrder.got.LineItems[0].SKUID)
	require.NotNil(t, createOrder.got.SalesChannelID)
	require.Equal(t, "channel-1", *createOrder.got.SalesChannelID)
	require.Equal(t, omsDomain.SalesOrderSourceInternal, createOrder.got.Source)
	require.Equal(t, omsDomain.SalesOrderStatusPending, createOrder.got.Status)
	require.Equal(t, "existing-customer-1", *createOrder.got.CustomerID)
	require.Equal(t, int64(1000), createOrder.got.TotalMinorUnits)
	require.NotNil(t, cartRepo.updated)
	require.NotNil(t, cartRepo.updated.CheckedOutAt)
	require.Contains(t, outboxRepo.subjects, "storefront.order.placed")
	require.Equal(t, "EUR", result.CurrencyCode)
	require.Nil(t, result.PaymentURL)
}

func TestCheckoutCart_createsAGuestCustomerWhenEmailIsNew(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	createOrder := &fakeSalesOrderCreator{}
	uc := newCheckoutUseCase(t, checkoutTxnCount, cartRepo, &fakeCustomerRepo{}, &fakeCustomerCreator{}, createOrder, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.NoError(t, err)
	require.Equal(t, "new-customer-1", *createOrder.got.CustomerID)
}

func TestCheckoutCart_appliesTaxRateForShippingCountryOnTopOfExclusivePrice(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	createOrder := &fakeSalesOrderCreator{}
	taxClassRepo := &fakeTaxClassRepo{out: []taxDomain.TaxClass{{ID: "standard", IsDefault: true}}}
	taxRateRepo := &fakeTaxRateRepo{out: []taxDomain.TaxRate{{TaxClassID: "standard", Country: "NL", RateBasisPoints: 2100}}}
	uc := newCheckoutUseCase(t, checkoutTxnCount, cartRepo, customerRepo, &fakeCustomerCreator{}, createOrder, taxClassRepo, taxRateRepo, &fakeOutboxRepo{}, nil)

	result, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.NoError(t, err)
	// subtotal 1000 * 21% = 210
	require.Equal(t, int64(210), createOrder.got.TaxTotalMinorUnits)
	require.Equal(t, int64(1210), createOrder.got.TotalMinorUnits)
	require.Equal(t, int64(1210), result.TotalMinorUnits)
}

func TestCheckoutCart_generatesAPaymentLinkWhenChannelHasOneConfigured(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	paymentGen := &fakePaymentLinkGenerator{link: &integrationsDomain.PaymentLink{URL: "https://pay.example/abc"}, ok: true}
	uc := newCheckoutUseCase(t, checkoutTxnCount, cartRepo, customerRepo, &fakeCustomerCreator{}, &fakeSalesOrderCreator{}, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, paymentGen)

	result, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.NoError(t, err)
	require.NotNil(t, result.PaymentURL)
	require.Equal(t, "https://pay.example/abc", *result.PaymentURL)
}

func TestCheckoutCart_neverFailsWhenPaymentLinkGenerationErrors(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	paymentGen := &fakePaymentLinkGenerator{err: context.DeadlineExceeded}
	uc := newCheckoutUseCase(t, checkoutTxnCount, cartRepo, customerRepo, &fakeCustomerCreator{}, &fakeSalesOrderCreator{}, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, paymentGen)

	result, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.NoError(t, err)
	require.Nil(t, result.PaymentURL)
}

func TestCheckoutCart_rejectsEmptyCart(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": {ID: "cart-1", SalesChannelID: "channel-1"}}}
	uc := newCheckoutUseCase(t, 1, cartRepo, &fakeCustomerRepo{}, &fakeCustomerCreator{}, &fakeSalesOrderCreator{}, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.ErrorIs(t, err, storefrontDomain.ErrCartEmpty)
}

func TestCheckoutCart_rejectsAlreadyCheckedOutCart(t *testing.T) {
	cart := openCartWithOneItem()
	cart.CheckedOutAt = timePtr()
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": cart}}
	uc := newCheckoutUseCase(t, 1, cartRepo, &fakeCustomerRepo{}, &fakeCustomerCreator{}, &fakeSalesOrderCreator{}, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.ErrorIs(t, err, storefrontDomain.ErrCartAlreadyCheckedOut)
}

func TestCheckoutCart_rejectsUnknownCart(t *testing.T) {
	uc := newCheckoutUseCase(t, 1, &fakeCartRepo{}, &fakeCustomerRepo{}, &fakeCustomerCreator{}, &fakeSalesOrderCreator{}, &fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), "unknown-token", validCheckoutInput())

	require.ErrorIs(t, err, storefrontDomain.ErrCartNotFound)
}
