package whatsapp_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

const testPhone = "31612345678"

// orderFlowChannelRepo backs every OrderFlowUseCase test's resolveWarehouse
// call -- an active WhatsApp channel with a DefaultWarehouseID already
// configured, so only the dedicated warehouse-error test below needs to
// think about it.
func orderFlowChannelRepo() *fakeSalesChannelRepo {
	whID := "wh-1"
	return &fakeSalesChannelRepo{channels: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp, Active: true, DefaultWarehouseID: &whID},
	}}
}

func newOrderFlowUseCase(t *testing.T, n int, convRepo *fakeConversationRepo, lookup *fakeProductInfoLookup, orderCreator *fakeSalesOrderCreator, customerCreator *fakeCustomerCreator) *appwhatsapp.OrderFlowUseCase {
	t.Helper()
	if convRepo == nil {
		convRepo = &fakeConversationRepo{}
	}
	if lookup == nil {
		lookup = &fakeProductInfoLookup{}
	}
	if orderCreator == nil {
		orderCreator = &fakeSalesOrderCreator{}
	}
	if customerCreator == nil {
		customerCreator = &fakeCustomerCreator{}
	}
	return appwhatsapp.NewOrderFlowUseCase(newMockGormDB(t, n), convRepo, orderFlowChannelRepo(), lookup, orderCreator, customerCreator, nil, &eventrecordtest.Outbox{})
}

func widgetInfo() appwhatsapp.ProductInfo {
	return appwhatsapp.ProductInfo{SKUID: "sku-1", SKUCode: "ABC123", ProductName: "Widget", StockQuantity: 10, PriceMinorUnits: 1999, CurrencyCode: "EUR"}
}

// ── ORDER (add to cart) ──────────────────────────────────────────────────────

func TestOrderFlow_orderCommand_addsToCartAndReplies(t *testing.T) {
	convRepo := &fakeConversationRepo{}
	lookup := &fakeProductInfoLookup{info: ptr(widgetInfo())}
	uc := newOrderFlowUseCase(t, 2, convRepo, lookup, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "ORDER ABC123 2")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "Added 2x Widget")
	require.Len(t, convRepo.saved, 1)
	require.Len(t, convRepo.saved[0].Cart, 1)
	assert.Equal(t, int64(2), convRepo.saved[0].Cart[0].Quantity)
}

func TestOrderFlow_orderCommand_sameSKUTwice_mergesQuantity(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, Cart: []whatsappDomain.CartItem{{SKUID: "sku-1", SKUCode: "ABC123", ProductName: "Widget", Quantity: 1, PriceMinorUnits: 1999, CurrencyCode: "EUR"}}},
	}}
	lookup := &fakeProductInfoLookup{info: ptr(widgetInfo())}
	uc := newOrderFlowUseCase(t, 2, convRepo, lookup, nil, nil)

	_, handled, err := uc.Execute(context.Background(), testPhone, nil, "ORDER ABC123 2")

	require.NoError(t, err)
	assert.True(t, handled)
	require.Len(t, convRepo.saved[0].Cart, 1, "same SKU must merge into one line, not add a second")
	assert.Equal(t, int64(3), convRepo.saved[0].Cart[0].Quantity)
}

func TestOrderFlow_orderCommand_unknownSKU_repliesGracefullyWithoutSaving(t *testing.T) {
	convRepo := &fakeConversationRepo{}
	lookup := &fakeProductInfoLookup{err: appwhatsapp.ErrProductNotFound}
	uc := newOrderFlowUseCase(t, 1, convRepo, lookup, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "ORDER ZZZ 1")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "couldn't find")
	assert.Empty(t, convRepo.saved, "nothing to persist when the product lookup fails")
}

func TestOrderFlow_orderCommand_exceedsStock_rejectsWithoutSaving(t *testing.T) {
	convRepo := &fakeConversationRepo{}
	info := widgetInfo()
	info.StockQuantity = 1
	lookup := &fakeProductInfoLookup{info: &info}
	uc := newOrderFlowUseCase(t, 1, convRepo, lookup, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "ORDER ABC123 5")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "Only 1 in stock")
	assert.Empty(t, convRepo.saved)
}

func TestOrderFlow_orderCommand_malformed_repliesUsageWithoutLookup(t *testing.T) {
	convRepo := &fakeConversationRepo{}
	lookup := &fakeProductInfoLookup{}
	uc := newOrderFlowUseCase(t, 1, convRepo, lookup, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "ORDER ABC123 notanumber")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "ORDER <code> <quantity>")
}

// ── CART ─────────────────────────────────────────────────────────────────────

func TestOrderFlow_cartCommand_empty(t *testing.T) {
	uc := newOrderFlowUseCase(t, 1, nil, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CART")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "empty")
}

func TestOrderFlow_cartCommand_showsItemsAndTotal(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, Cart: []whatsappDomain.CartItem{{SKUID: "sku-1", SKUCode: "ABC123", ProductName: "Widget", Quantity: 2, PriceMinorUnits: 1000, CurrencyCode: "EUR"}}},
	}}
	uc := newOrderFlowUseCase(t, 1, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CART")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "Widget")
	assert.Contains(t, reply, "20.00 EUR")
}

// ── CHECKOUT ─────────────────────────────────────────────────────────────────

func nonEmptyCartConv() *fakeConversationRepo {
	return &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, Cart: []whatsappDomain.CartItem{{SKUID: "sku-1", SKUCode: "ABC123", ProductName: "Widget", Quantity: 1, PriceMinorUnits: 1999, CurrencyCode: "EUR"}}},
	}}
}

func TestOrderFlow_checkout_emptyCart_rejects(t *testing.T) {
	uc := newOrderFlowUseCase(t, 1, nil, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CHECKOUT")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "empty")
}

func TestOrderFlow_checkout_knownCompleteCustomer_jumpsStraightToConfirmation(t *testing.T) {
	convRepo := nonEmptyCartConv()
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	reply, handled, err := uc.Execute(context.Background(), testPhone, customer, "CHECKOUT")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "YES")
	assert.Contains(t, reply, "Main St 1")
	assert.Equal(t, whatsappDomain.ConversationStateAwaitingOrderConfirmation, convRepo.saved[len(convRepo.saved)-1].State)
}

func TestOrderFlow_checkout_unknownCustomer_startsNameCapture(t *testing.T) {
	convRepo := nonEmptyCartConv()
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CHECKOUT")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "full name")
	assert.Equal(t, whatsappDomain.ConversationStateAwaitingNewCustomerName, convRepo.saved[len(convRepo.saved)-1].State)
}

func TestOrderFlow_checkout_knownCustomerMissingEmail_startsCapture(t *testing.T) {
	convRepo := nonEmptyCartConv()
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	_, handled, err := uc.Execute(context.Background(), testPhone, customer, "CHECKOUT")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, whatsappDomain.ConversationStateAwaitingNewCustomerName, convRepo.saved[len(convRepo.saved)-1].State, "an incomplete profile goes through capture, same as an unknown customer")
}

// ── new-customer capture ─────────────────────────────────────────────────────

func TestOrderFlow_captureName_valid_movesToEmail(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerName},
	}}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "Jane Doe")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "email")
	last := convRepo.saved[len(convRepo.saved)-1]
	assert.Equal(t, whatsappDomain.ConversationStateAwaitingNewCustomerEmail, last.State)
	assert.Equal(t, "Jane Doe", last.PendingCustomerName)
}

func TestOrderFlow_captureName_blank_reprompts(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerName},
	}}
	uc := newOrderFlowUseCase(t, 1, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "   ")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "full name")
	assert.Empty(t, convRepo.saved, "a blank answer must not persist or advance state")
}

func TestOrderFlow_captureEmail_invalid_reprompts(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerEmail, PendingCustomerName: "Jane"},
	}}
	uc := newOrderFlowUseCase(t, 1, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "not-an-email")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "valid email")
	assert.Empty(t, convRepo.saved)
}

func TestOrderFlow_captureEmail_valid_movesToAddress(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerEmail, PendingCustomerName: "Jane"},
	}}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "jane@example.com")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "shipping address")
	last := convRepo.saved[len(convRepo.saved)-1]
	assert.Equal(t, whatsappDomain.ConversationStateAwaitingNewCustomerAddress, last.State)
	assert.Equal(t, "jane@example.com", last.PendingCustomerEmail)
}

func TestOrderFlow_captureAddress_valid_movesToConfirmation(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {
			PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerAddress,
			PendingCustomerName: "Jane", PendingCustomerEmail: "jane@example.com",
			Cart: []whatsappDomain.CartItem{{SKUID: "sku-1", SKUCode: "ABC123", ProductName: "Widget", Quantity: 1, PriceMinorUnits: 1999, CurrencyCode: "EUR"}},
		},
	}}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "Main St 1, Amsterdam, 1000AB, NL")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "YES")
	assert.Contains(t, reply, "Jane")
	assert.Contains(t, reply, "jane@example.com")
	last := convRepo.saved[len(convRepo.saved)-1]
	assert.Equal(t, whatsappDomain.ConversationStateAwaitingOrderConfirmation, last.State)
	assert.Equal(t, "Main St 1, Amsterdam, 1000AB, NL", last.PendingCustomerAddress)
}

// ── confirmation ─────────────────────────────────────────────────────────────

func awaitingConfirmationConv(withPending bool) *fakeConversationRepo {
	conv := whatsappDomain.Conversation{
		PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingOrderConfirmation,
		Cart: []whatsappDomain.CartItem{{SKUID: "sku-1", SKUCode: "ABC123", ProductName: "Widget", Quantity: 2, PriceMinorUnits: 1999, CurrencyCode: "EUR"}},
	}
	if withPending {
		conv.PendingCustomerName, conv.PendingCustomerEmail, conv.PendingCustomerAddress = "Jane Doe", "jane@example.com", "Main St 1"
	}
	return &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{testPhone: conv}}
}

func TestOrderFlow_confirmation_yesNewCustomer_createsCustomerAndOrder(t *testing.T) {
	convRepo := awaitingConfirmationConv(true)
	orderCreator := &fakeSalesOrderCreator{}
	customerCreator := &fakeCustomerCreator{}
	uc := newOrderFlowUseCase(t, 3, convRepo, nil, orderCreator, customerCreator)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "YES")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.True(t, customerCreator.called)
	assert.Equal(t, "Jane", customerCreator.in.FirstName)
	assert.Equal(t, "Doe", customerCreator.in.LastName)
	assert.Equal(t, "jane@example.com", customerCreator.in.Email)
	assert.Equal(t, testPhone, customerCreator.in.PhoneE164)

	require.True(t, orderCreator.called)
	assert.Equal(t, omsDomain.SalesOrderSourceWhatsApp, orderCreator.in.Source)
	assert.Equal(t, omsDomain.SalesOrderStatusPending, orderCreator.in.Status)
	require.NotNil(t, orderCreator.in.CustomerID)
	assert.Equal(t, "new-cust-1", *orderCreator.in.CustomerID)
	require.Len(t, orderCreator.in.LineItems, 1)
	assert.Equal(t, int64(2), orderCreator.in.LineItems[0].Quantity)
	assert.Equal(t, int64(3998), orderCreator.in.TotalMinorUnits)

	assert.Contains(t, reply, "placed")
	last := convRepo.saved[len(convRepo.saved)-1]
	assert.Equal(t, whatsappDomain.ConversationStateIdle, last.State)
	assert.Empty(t, last.Cart, "cart must be cleared after placing the order")
}

func TestOrderFlow_confirmation_yesKnownCustomer_doesNotCreateCustomer(t *testing.T) {
	convRepo := awaitingConfirmationConv(false)
	orderCreator := &fakeSalesOrderCreator{}
	customerCreator := &fakeCustomerCreator{}
	uc := newOrderFlowUseCase(t, 3, convRepo, nil, orderCreator, customerCreator)
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	_, handled, err := uc.Execute(context.Background(), testPhone, customer, "yes")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.False(t, customerCreator.called, "an already-known customer must never be recreated")
	require.True(t, orderCreator.called)
	require.NotNil(t, orderCreator.in.CustomerID)
	assert.Equal(t, "cust-1", *orderCreator.in.CustomerID)
}

func TestOrderFlow_confirmation_no_cancelsOrder(t *testing.T) {
	convRepo := awaitingConfirmationConv(true)
	orderCreator := &fakeSalesOrderCreator{}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, orderCreator, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "NO")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.False(t, orderCreator.called)
	assert.Contains(t, reply, "cancelled")
	last := convRepo.saved[len(convRepo.saved)-1]
	assert.Equal(t, whatsappDomain.ConversationStateIdle, last.State)
	assert.Empty(t, last.Cart)
}

func TestOrderFlow_confirmation_garbage_repromptsWithoutChangingState(t *testing.T) {
	convRepo := awaitingConfirmationConv(true)
	uc := newOrderFlowUseCase(t, 1, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "maybe")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "YES")
	assert.Empty(t, convRepo.saved, "an unrecognized answer must not persist or change state")
}

// TestOrderFlow_confirmation_channelHasNoDefaultWarehouse_failsWithClearError
// guards the live-discovered bug this fix addresses: sales_orders.warehouse_id
// is a NOT NULL FK, so a WhatsApp channel with no DefaultWarehouseID
// configured must fail with ErrChannelWarehouseNotConfigured rather than let
// a raw invalid-uuid DB error surface.
func TestOrderFlow_confirmation_channelHasNoDefaultWarehouse_failsWithClearError(t *testing.T) {
	convRepo := awaitingConfirmationConv(false)
	channelRepo := &fakeSalesChannelRepo{channels: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWhatsApp, Active: true},
	}}
	uc := appwhatsapp.NewOrderFlowUseCase(newMockGormDB(t, 2), convRepo, channelRepo, &fakeProductInfoLookup{}, &fakeSalesOrderCreator{}, &fakeCustomerCreator{}, nil, &eventrecordtest.Outbox{})
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	_, _, err := uc.Execute(context.Background(), testPhone, customer, "YES")

	assert.ErrorIs(t, err, appwhatsapp.ErrChannelWarehouseNotConfigured)
}

func TestOrderFlow_confirmation_orderCreationFails_propagatesError(t *testing.T) {
	convRepo := awaitingConfirmationConv(false)
	orderCreator := &fakeSalesOrderCreator{err: errors.New("db down")}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, orderCreator, nil)
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	_, _, err := uc.Execute(context.Background(), testPhone, customer, "YES")

	require.Error(t, err, "a failed order creation must surface as a dispatch error, not a false success reply")
}

// ── CANCEL ───────────────────────────────────────────────────────────────────

func TestOrderFlow_cancel_clearsStateAndCart(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {
			PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerEmail,
			Cart: []whatsappDomain.CartItem{{SKUID: "sku-1", Quantity: 1}},
		},
	}}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CANCEL")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "cancelled")
	last := convRepo.saved[len(convRepo.saved)-1]
	assert.Equal(t, whatsappDomain.ConversationStateIdle, last.State)
	assert.Empty(t, last.Cart)
}

func TestOrderFlow_cancel_idleAndEmpty_saysNothingToCancel(t *testing.T) {
	uc := newOrderFlowUseCase(t, 2, nil, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CANCEL")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "Nothing to cancel")
}

func TestOrderFlow_cancel_interruptsCaptureFlowInsteadOfBeingTreatedAsAnAnswer(t *testing.T) {
	convRepo := &fakeConversationRepo{byPhone: map[string]whatsappDomain.Conversation{
		testPhone: {PhoneE164: testPhone, State: whatsappDomain.ConversationStateAwaitingNewCustomerName},
	}}
	uc := newOrderFlowUseCase(t, 2, convRepo, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "CANCEL")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "cancelled")
	assert.Equal(t, whatsappDomain.ConversationStateIdle, convRepo.saved[len(convRepo.saved)-1].State)
}

// ── unhandled ────────────────────────────────────────────────────────────────

func TestOrderFlow_idleUnrecognizedCommand_returnsUnhandled(t *testing.T) {
	uc := newOrderFlowUseCase(t, 1, nil, nil, nil, nil)

	reply, handled, err := uc.Execute(context.Background(), testPhone, nil, "STOCK ABC123")

	require.NoError(t, err)
	assert.False(t, handled, "STOCK is HandleInboundMessageUseCase's own command, not this flow's")
	assert.Empty(t, reply)
}

func ptr[T any](v T) *T { return &v }

// ── payment link generation on order placement ──────────────────────────────

func TestOrderFlow_confirmation_yes_paymentMethodConfigured_includesLinkInReply(t *testing.T) {
	convRepo := awaitingConfirmationConv(false)
	paymentGen := &fakePaymentLinkGenerator{ok: true, link: &integrationsDomain.PaymentLink{URL: "https://checkout.stripe.com/pay/cs_1"}}
	uc := appwhatsapp.NewOrderFlowUseCase(newMockGormDB(t, 3), convRepo, orderFlowChannelRepo(), &fakeProductInfoLookup{}, &fakeSalesOrderCreator{}, &fakeCustomerCreator{}, paymentGen, &eventrecordtest.Outbox{})
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	reply, handled, err := uc.Execute(context.Background(), testPhone, customer, "YES")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "placed")
	assert.Contains(t, reply, "https://checkout.stripe.com/pay/cs_1")
}

func TestOrderFlow_confirmation_yes_noPaymentMethodConfigured_omitsLink(t *testing.T) {
	convRepo := awaitingConfirmationConv(false)
	paymentGen := &fakePaymentLinkGenerator{ok: false}
	uc := appwhatsapp.NewOrderFlowUseCase(newMockGormDB(t, 3), convRepo, orderFlowChannelRepo(), &fakeProductInfoLookup{}, &fakeSalesOrderCreator{}, &fakeCustomerCreator{}, paymentGen, &eventrecordtest.Outbox{})
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	reply, handled, err := uc.Execute(context.Background(), testPhone, customer, "YES")

	require.NoError(t, err)
	assert.True(t, handled)
	assert.Contains(t, reply, "placed")
	assert.NotContains(t, reply, "http")
}

func TestOrderFlow_confirmation_yes_paymentLinkGenerationFails_stillSucceedsWithoutLink(t *testing.T) {
	convRepo := awaitingConfirmationConv(false)
	paymentGen := &fakePaymentLinkGenerator{err: errors.New("stripe unreachable")}
	uc := appwhatsapp.NewOrderFlowUseCase(newMockGormDB(t, 3), convRepo, orderFlowChannelRepo(), &fakeProductInfoLookup{}, &fakeSalesOrderCreator{}, &fakeCustomerCreator{}, paymentGen, &eventrecordtest.Outbox{})
	customer := &customersDomain.Customer{ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", Shipping: customersDomain.Address{Address1: "Main St 1"}}

	reply, handled, err := uc.Execute(context.Background(), testPhone, customer, "YES")

	require.NoError(t, err, "a payment-link failure must never fail order placement -- retrying this handler would place a duplicate order")
	assert.True(t, handled)
	assert.Contains(t, reply, "placed")
}
