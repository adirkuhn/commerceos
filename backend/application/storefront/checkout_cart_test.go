package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
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

// fakeConnectionRepo/fakeShippingSettingsRepo/fakeDeliveryPriceEstimator
// back CheckoutCartUseCase's optional shipping-estimate dependencies --
// newCheckoutUseCase passes nil for all three (estimateShipping's nil
// check short-circuits before ever touching them), so only the shipping-
// specific tests below construct the use case with real fakes here.
type fakeConnectionRepo struct {
	byTenant []integrationsDomain.Connection
	err      error
}

func (f *fakeConnectionRepo) Create(context.Context, *integrationsDomain.Connection) error {
	return nil
}
func (f *fakeConnectionRepo) FindByID(context.Context, string) (*integrationsDomain.Connection, error) {
	return nil, nil
}
func (f *fakeConnectionRepo) ListByTenant(context.Context) ([]integrationsDomain.Connection, error) {
	return f.byTenant, f.err
}
func (f *fakeConnectionRepo) Update(context.Context, *integrationsDomain.Connection) error {
	return nil
}
func (f *fakeConnectionRepo) Delete(context.Context, string) error { return nil }

type fakeShippingSettingsRepo struct {
	out *shippingDomain.ShippingConnectionSettings
	err error
}

func (f *fakeShippingSettingsRepo) FindByConnectionID(context.Context, string) (*shippingDomain.ShippingConnectionSettings, error) {
	return f.out, f.err
}
func (f *fakeShippingSettingsRepo) Upsert(context.Context, *shippingDomain.ShippingConnectionSettings) error {
	return nil
}

type fakeDeliveryPriceEstimator struct {
	out []shippingDomain.DeliveryPriceOption
	err error

	gotInput appshipping.GetDeliveryPriceEstimateInput
}

func (f *fakeDeliveryPriceEstimator) Execute(_ context.Context, in appshipping.GetDeliveryPriceEstimateInput) ([]shippingDomain.DeliveryPriceOption, error) {
	f.gotInput = in
	return f.out, f.err
}

func newCheckoutUseCase(t *testing.T, n int, cartRepo *fakeCartRepo, customerRepo *fakeCustomerRepo, createCustomer *fakeCustomerCreator, createOrder *fakeSalesOrderCreator, taxClassRepo *fakeTaxClassRepo, taxRateRepo *fakeTaxRateRepo, outboxRepo *fakeOutboxRepo, paymentGen *fakePaymentLinkGenerator) *appstorefront.CheckoutCartUseCase {
	return appstorefront.NewCheckoutCartUseCase(
		newMockGormDB(t, n), cartRepo, customerRepo, createCustomer, createOrder, taxClassRepo, taxRateRepo, outboxRepo, paymentGen,
		nil, nil, nil,
	)
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

// shippingCheckoutTxnCount adds one transaction over checkoutTxnCount --
// estimateShipping's own read-only lookup of the tenant's MyParcel
// connection + its ShippingConnectionSettings.
const shippingCheckoutTxnCount = checkoutTxnCount + 1

func TestCheckoutCart_addsShippingCostFromTheCheapestMyParcelOption(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	createOrder := &fakeSalesOrderCreator{}
	connRepo := &fakeConnectionRepo{byTenant: []integrationsDomain.Connection{{ID: "conn-1", Provider: integrationsDomain.ProviderMyParcel}}}
	settingsRepo := &fakeShippingSettingsRepo{out: &shippingDomain.ShippingConnectionSettings{ConnectionID: "conn-1", DefaultCarrier: shippingDomain.CarrierPostNL}}
	estimator := &fakeDeliveryPriceEstimator{out: []shippingDomain.DeliveryPriceOption{
		{PriceComment: "Evening", AmountMinorUnits: 895, Currency: "EUR"},
		{PriceComment: "Standard", AmountMinorUnits: 495, Currency: "EUR"},
	}}
	uc := appstorefront.NewCheckoutCartUseCase(
		newMockGormDB(t, shippingCheckoutTxnCount), cartRepo, customerRepo, &fakeCustomerCreator{}, createOrder,
		&fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil, connRepo, settingsRepo, estimator,
	)

	addr := appstorefront.CheckoutAddressInput{FirstName: "A", LastName: "B", Address1: "Damrak 1", City: "Amsterdam", Postcode: "1012 LG", Country: "NL"}
	result, err := uc.Execute(context.Background(), "token-1", appstorefront.CheckoutInput{Email: "shopper@example.com", Billing: addr, Shipping: addr})

	require.NoError(t, err)
	require.Equal(t, int64(495), createOrder.got.ShippingTotalMinorUnits)
	require.Equal(t, "myparcel", createOrder.got.ShippingMethod)
	require.Equal(t, "Standard", createOrder.got.ShippingMethodTitle)
	require.Equal(t, int64(1495), createOrder.got.TotalMinorUnits) // 1000 subtotal + 495 shipping, no tax configured
	require.Equal(t, int64(1495), result.TotalMinorUnits)
	require.Equal(t, "conn-1", estimator.gotInput.ConnectionID)
	require.Equal(t, "NL", estimator.gotInput.CC)
	require.Equal(t, "1012 LG", estimator.gotInput.PostalCode)
	require.Equal(t, "1", estimator.gotInput.Number)
	require.Equal(t, shippingDomain.CarrierPostNL, estimator.gotInput.Carrier)
}

func TestCheckoutCart_skipsShippingEstimateWhenNoHouseNumberCanBeParsed(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	createOrder := &fakeSalesOrderCreator{}
	connRepo := &fakeConnectionRepo{byTenant: []integrationsDomain.Connection{{ID: "conn-1", Provider: integrationsDomain.ProviderMyParcel}}}
	settingsRepo := &fakeShippingSettingsRepo{}
	estimator := &fakeDeliveryPriceEstimator{out: []shippingDomain.DeliveryPriceOption{{PriceComment: "Standard", AmountMinorUnits: 495, Currency: "EUR"}}}
	uc := appstorefront.NewCheckoutCartUseCase(
		newMockGormDB(t, checkoutTxnCount), cartRepo, customerRepo, &fakeCustomerCreator{}, createOrder,
		&fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil, connRepo, settingsRepo, estimator,
	)

	addr := appstorefront.CheckoutAddressInput{FirstName: "A", LastName: "B", Address1: "Damrak", City: "Amsterdam", Postcode: "1012 LG", Country: "NL"}
	_, err := uc.Execute(context.Background(), "token-1", appstorefront.CheckoutInput{Email: "shopper@example.com", Billing: addr, Shipping: addr})

	require.NoError(t, err)
	require.Zero(t, createOrder.got.ShippingTotalMinorUnits)
	require.Empty(t, createOrder.got.ShippingMethod)
}

func TestCheckoutCart_skipsShippingEstimateForAnUnsupportedCountry(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	createOrder := &fakeSalesOrderCreator{}
	connRepo := &fakeConnectionRepo{byTenant: []integrationsDomain.Connection{{ID: "conn-1", Provider: integrationsDomain.ProviderMyParcel}}}
	estimator := &fakeDeliveryPriceEstimator{err: shippingDomain.ErrDeliveryEstimateUnsupportedCountry}
	uc := appstorefront.NewCheckoutCartUseCase(
		newMockGormDB(t, checkoutTxnCount), cartRepo, customerRepo, &fakeCustomerCreator{}, createOrder,
		&fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil, connRepo, &fakeShippingSettingsRepo{}, estimator,
	)

	addr := appstorefront.CheckoutAddressInput{FirstName: "A", LastName: "B", Address1: "Main St 1", City: "Berlin", Postcode: "10115", Country: "DE"}
	_, err := uc.Execute(context.Background(), "token-1", appstorefront.CheckoutInput{Email: "shopper@example.com", Billing: addr, Shipping: addr})

	require.NoError(t, err)
	require.Zero(t, createOrder.got.ShippingTotalMinorUnits)
}

func TestCheckoutCart_skipsShippingEstimateWhenNoMyParcelConnectionExists(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	customerRepo := &fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "existing-customer-1"}}}
	createOrder := &fakeSalesOrderCreator{}
	estimator := &fakeDeliveryPriceEstimator{out: []shippingDomain.DeliveryPriceOption{{PriceComment: "Standard", AmountMinorUnits: 495, Currency: "EUR"}}}
	uc := appstorefront.NewCheckoutCartUseCase(
		newMockGormDB(t, shippingCheckoutTxnCount), cartRepo, customerRepo, &fakeCustomerCreator{}, createOrder,
		&fakeTaxClassRepo{}, &fakeTaxRateRepo{}, &fakeOutboxRepo{}, nil, &fakeConnectionRepo{}, &fakeShippingSettingsRepo{}, estimator,
	)

	_, err := uc.Execute(context.Background(), "token-1", validCheckoutInput())

	require.NoError(t, err)
	require.Zero(t, createOrder.got.ShippingTotalMinorUnits)
}
