package moneybird_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/oms"
)

type fakeOrders struct {
	order    oms.SalesOrder
	lines    []oms.SalesOrderLineItem
	payments oms.SalesOrderPayments
}

func (f *fakeOrders) FindByID(_ context.Context, id string) (*oms.SalesOrder, error) {
	if id != f.order.ID {
		return nil, oms.ErrSalesOrderNotFound
	}
	o := f.order
	return &o, nil
}
type fakeLines struct{ orders *fakeOrders }

func (f fakeLines) ListBySalesOrder(_ context.Context, _ string) ([]oms.SalesOrderLineItem, error) {
	return f.orders.lines, nil
}

type fakePayments struct{ orders *fakeOrders }

func (f fakePayments) ListBySalesOrder(_ context.Context, _ string) (oms.SalesOrderPayments, error) {
	return f.orders.payments, nil
}

type fakeCustomers struct{ byID map[string]customersDomain.Customer }

func (f *fakeCustomers) FindByID(_ context.Context, id string) (*customersDomain.Customer, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, customersDomain.ErrCustomerNotFound
	}
	return &c, nil
}

// A paid bank-transfer Woo order: 2 x Farofa (9%) + PostNL shipping.
func paidOrder() *fakeOrders {
	paid := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	return &fakeOrders{
		order: oms.SalesOrder{
			ID: "so-1", TenantID: "tenant-1", Number: "1234", CurrencyCode: "EUR", Status: oms.SalesOrderStatusProcessing,
			PaymentMethod: "bacs", DatePaid: &paid, OccurredAt: time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC),
			ShippingTotalMinorUnits: 495, ShippingMethodTitle: "PostNL", TotalMinorUnits: 698 + 599, TaxTotalMinorUnits: 58 + 104,
			Billing: oms.Address{FirstName: "Ana", LastName: "Souza", Company: "Mercado Lusitano B.V.", Email: "ana@example.nl", Address1: "Kerkstraat 10", Postcode: "1017GB", City: "Amsterdam", Country: "NL"},
		},
		lines: []oms.SalesOrderLineItem{{Name: "Farofa", Quantity: 2, TotalMinorUnits: 640, TotalTaxMinorUnits: 58}},
	}
}

type orderHarness struct {
	orders   *fakeOrders
	client   *fakeClient
	docs     *fakeDocuments
	contacts *fakeContacts
	settings *fakeSettings
	outbox   *eventrecordtest.Outbox
	uc       *appmb.SendSalesOrderUseCase
}

func newOrderHarness(t *testing.T, orders *fakeOrders, docs ...mb.Document) *orderHarness {
	h := &orderHarness{
		orders: orders, client: &fakeClient{}, docs: newFakeDocuments(docs...), contacts: &fakeContacts{},
		settings: newFakeSettings(configuredSettings()), outbox: &eventrecordtest.Outbox{},
	}
	h.uc = appmb.NewSendSalesOrderUseCase(mockDB(t, 8), appmb.SendSalesOrderDeps{
		Connections: connected(t, "tok"), Settings: h.settings, Documents: h.docs, Contacts: h.contacts,
		Orders: orders, Lines: fakeLines{orders}, Payments: fakePayments{orders}, Customers: &fakeCustomers{}, Factory: &fakeFactory{client: h.client}, Outbox: h.outbox, Secret: testSecret,
	})
	return h
}

func TestSendSalesOrder_booksAPaidOrderWithItsPayment(t *testing.T) {
	h := newOrderHarness(t, paidOrder())

	doc, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	require.Len(t, h.client.createdInvoices, 1)
	inv := h.client.createdInvoices[0]
	assert.Equal(t, "1234", inv.Reference)
	assert.Equal(t, "2026-10-28", inv.Date)
	assert.Equal(t, "EUR", inv.Currency)
	assert.True(t, inv.PricesInclTax)
	assert.Equal(t, []mb.DocumentLine{
		{Description: "Farofa", Amount: "2", PriceMinorUnits: 349, TaxRateID: "tr-9", LedgerAccountID: "led-1"},
		{Description: "Shipping: PostNL", Amount: "1", PriceMinorUnits: 599, TaxRateID: "tr-21", LedgerAccountID: "led-1"},
	}, inv.Lines)
	// A new contact for the billing company.
	require.Len(t, h.client.createdContacts, 1)
	assert.Equal(t, "Mercado Lusitano B.V.", h.client.createdContacts[0].CompanyName)
	assert.Equal(t, "ana@example.nl", h.client.createdContacts[0].Email)
	assert.Equal(t, "mb-contact-new", inv.ContactID)
	// Paid by bank transfer, on the day it was paid.
	assert.Equal(t, []mb.Payment{{Date: "2026-10-28", Price: "12.97", Method: "bank_transfer"}}, h.client.payments)

	assert.Equal(t, mb.DocumentSynced, doc.Status)
	assert.Equal(t, "inv-new", doc.MoneybirdID)
	assert.True(t, doc.PaymentRegistered)
	assert.Equal(t, []string{"integrations.moneybird_document.synced"}, h.outbox.Subjects())
}

func TestSendSalesOrder_unpaidOrderIsBookedWithoutPayment(t *testing.T) {
	orders := paidOrder()
	orders.order.Status, orders.order.DatePaid = oms.SalesOrderStatusPending, nil
	h := newOrderHarness(t, orders)

	doc, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Len(t, h.client.createdInvoices, 1)
	assert.Empty(t, h.client.payments)
	assert.False(t, doc.PaymentRegistered)
}

// Another service already booked this order in Moneybird: never a second invoice.
func TestSendSalesOrder_refusesWhenMoneybirdAlreadyHasTheReference(t *testing.T) {
	h := newOrderHarness(t, paidOrder())
	h.client.existingInvoice = &mb.RemoteDocument{ID: "theirs"}

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	assert.True(t, errors.Is(err, mb.ErrDuplicateReference))
	assert.Empty(t, h.client.createdInvoices)
	assert.Equal(t, mb.DocumentFailed, h.docs.last().Status)
	assert.Empty(t, h.docs.last().MoneybirdID, "not ours -- never linked")
	assert.Equal(t, []string{"integrations.moneybird_document.sync_failed"}, h.outbox.Subjects())
}

// Sending a booked order again changes nothing.
func TestSendSalesOrder_alreadyBookedIsANoOp(t *testing.T) {
	booked := mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", MoneybirdID: "inv-1", Status: mb.DocumentSynced, PaymentRegistered: true}
	h := newOrderHarness(t, paidOrder(), booked)

	doc, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Equal(t, "inv-1", doc.MoneybirdID)
	assert.Empty(t, h.client.createdInvoices)
	assert.Empty(t, h.client.payments)
	assert.Empty(t, h.outbox.Subjects())
}

// Booked unpaid, paid since: only the payment is added.
func TestSendSalesOrder_bookedBeforePaymentRegistersThePaymentOnly(t *testing.T) {
	booked := mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", MoneybirdID: "inv-1", Reference: "1234", Status: mb.DocumentSynced}
	h := newOrderHarness(t, paidOrder(), booked)
	h.client.invoices = map[string]mb.RemoteDocument{"inv-1": {ID: "inv-1", State: "open", TotalUnpaid: "12.97"}}

	doc, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Empty(t, h.client.createdInvoices)
	require.Len(t, h.client.payments, 1)
	assert.True(t, doc.PaymentRegistered)
}

// The invoice was created but the payment failed: a retry resumes it, never a second invoice.
func TestSendSalesOrder_failedPaymentKeepsTheInvoiceForTheRetry(t *testing.T) {
	h := newOrderHarness(t, paidOrder())
	h.client.paymentErr = mb.ErrRequestFailed

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	assert.Error(t, err)
	failed := h.docs.last()
	assert.Equal(t, mb.DocumentFailed, failed.Status)
	assert.Equal(t, "inv-new", failed.MoneybirdID)

	h.client.paymentErr = nil
	h.client.invoices = map[string]mb.RemoteDocument{"inv-new": {ID: "inv-new", State: "open", TotalUnpaid: "12.97"}}
	doc, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Len(t, h.client.createdInvoices, 1, "still the one invoice")
	assert.True(t, doc.PaymentRegistered)
}

// Moneybird already shows it paid (e.g. matched to the bank): don't pay twice.
func TestSendSalesOrder_invoiceAlreadyPaidInMoneybirdSkipsThePayment(t *testing.T) {
	booked := mb.Document{SourceType: mb.SourceSalesOrder, SourceID: "so-1", MoneybirdID: "inv-1", Status: mb.DocumentSynced}
	h := newOrderHarness(t, paidOrder(), booked)
	h.client.invoices = map[string]mb.RemoteDocument{"inv-1": {ID: "inv-1", State: "paid", TotalUnpaid: "0.0"}}

	doc, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Empty(t, h.client.payments)
	assert.True(t, doc.PaymentRegistered)
}

// Booking a paid order without its payment would show it unpaid -- refuse before creating anything.
func TestSendSalesOrder_paidWithAnUnmappedMethod(t *testing.T) {
	orders := paidOrder()
	orders.order.PaymentMethod = "stripe"
	h := newOrderHarness(t, orders)

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	assert.True(t, errors.Is(err, mb.ErrPaymentMethodNotMapped))
	assert.Empty(t, h.client.createdInvoices)
}

func TestSendSalesOrder_splitPaymentRegistersEachPart(t *testing.T) {
	orders := paidOrder()
	orders.payments = oms.SalesOrderPayments{{Method: "bacs", AmountMinorUnits: 1000}, {Method: "cash", AmountMinorUnits: 297}}
	h := newOrderHarness(t, orders)
	s := configuredSettings()
	s.PaymentMethods = append(s.PaymentMethods, mb.PaymentMethodMapping{Method: "cash", MoneybirdPaymentMethod: "cash"})
	h.settings.byConn["conn-mb"] = s

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Equal(t, []mb.Payment{
		{Date: "2026-10-28", Price: "10.00", Method: "bank_transfer"},
		{Date: "2026-10-28", Price: "2.97", Method: "cash"},
	}, h.client.payments)
}

func TestSendSalesOrder_unmappedVATRateNamesTheRate(t *testing.T) {
	orders := paidOrder()
	orders.lines[0].TotalTaxMinorUnits = 0 // 0% -- not mapped
	orders.order.TaxTotalMinorUnits = 104
	orders.order.TotalMinorUnits = 640 + 599
	h := newOrderHarness(t, orders)

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	assert.True(t, errors.Is(err, mb.ErrTaxRateNotMapped))
	assert.Contains(t, err.Error(), "0.00%")
	assert.Empty(t, h.client.createdInvoices)
	assert.Equal(t, mb.DocumentFailed, h.docs.last().Status)
}

func TestSendSalesOrder_notConfigured(t *testing.T) {
	h := newOrderHarness(t, paidOrder())
	s := configuredSettings()
	s.DefaultLedgerAccountID = ""
	h.settings.byConn["conn-mb"] = s

	_, err := h.uc.Execute(tenantCtx(), "so-1")
	assert.True(t, errors.Is(err, mb.ErrNotConfigured))
}

// The customer's Moneybird contact is found once, then remembered.
func TestSendSalesOrder_customerContactIsRemembered(t *testing.T) {
	orders := paidOrder()
	customerID := "cust-1"
	orders.order.CustomerID = &customerID
	h := newOrderHarness(t, orders)
	h.client.contacts = map[string]mb.Contact{"mb-ana": {ID: "mb-ana", Email: "ana@example.nl"}}

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	assert.Equal(t, "mb-ana", h.client.createdInvoices[0].ContactID)
	assert.Empty(t, h.client.createdContacts)
	assert.Equal(t, "mb-ana", h.contacts.byCustomer["cust-1"])
}

// Nothing to find or create a contact by: the shared guest contact.
func TestSendSalesOrder_anonymousOrderUsesTheGuestContact(t *testing.T) {
	orders := paidOrder()
	orders.order.Billing = oms.Address{}
	h := newOrderHarness(t, orders)

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	require.Len(t, h.client.createdContacts, 1)
	assert.Equal(t, "Guest customer", h.client.createdContacts[0].CompanyName)
	assert.Equal(t, "mb-contact-new", h.settings.byConn["conn-mb"].GuestContactID, "created once, kept for the next one")
}

// Moneybird needs the base-currency amount on a payment for a foreign-currency invoice.
func TestSendSalesOrder_foreignCurrencyPaymentCarriesTheBaseAmount(t *testing.T) {
	orders := paidOrder()
	orders.order.CurrencyCode = "USD"
	h := newOrderHarness(t, orders)
	h.client.administrations = []mb.Administration{{ID: "adm-1", Currency: "EUR"}}

	_, err := h.uc.Execute(tenantCtx(), "so-1")

	require.NoError(t, err)
	require.Len(t, h.client.payments, 1)
	assert.Equal(t, "12.11", h.client.payments[0].PriceBase)
}
