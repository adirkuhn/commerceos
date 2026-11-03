package moneybird

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// invoiceSource is what Moneybird shows an external sales invoice came from.
const invoiceSource = "CommerceOS"

type orderFinder interface {
	FindByID(ctx context.Context, id string) (*oms.SalesOrder, error)
}
type orderLineLister interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]oms.SalesOrderLineItem, error)
}
type orderPaymentLister interface {
	ListBySalesOrder(ctx context.Context, salesOrderID string) (oms.SalesOrderPayments, error)
}

type SendSalesOrderDeps struct {
	Connections connectionLister
	Settings    mb.SettingsRepository
	Documents   mb.DocumentRepository
	Contacts    mb.ContactRepository
	Orders      orderFinder
	Lines       orderLineLister
	Payments    orderPaymentLister
	Customers   customerReader
	Factory     mb.ClientFactory
	Outbox      eventrecord.Enqueuer
	Secret      string
}

// SendSalesOrderUseCase books a sales order in Moneybird as an external
// sales invoice (CommerceOS issued it; Moneybird only books it) and, when
// the order is paid, registers the payment on it -- like akantio_pos.
//
// Never books an order twice: an order already booked from here is a no-op
// (or only gets its payment, when it was booked unpaid), a half-finished
// earlier send resumes its invoice, and an invoice with the same reference
// that CommerceOS didn't create (another service booked the order) is
// refused with ErrDuplicateReference. A paid order whose payment method
// isn't mapped is refused before anything is created -- booked without its
// payment it would show as unpaid.
type SendSalesOrderUseCase struct {
	db       *gorm.DB
	deps     SendSalesOrderDeps
	recorder documentRecorder
	contacts contactResolver
}

func NewSendSalesOrderUseCase(db *gorm.DB, deps SendSalesOrderDeps) *SendSalesOrderUseCase {
	return &SendSalesOrderUseCase{
		db: db, deps: deps,
		recorder: documentRecorder{db: db, docs: deps.Documents, outbox: deps.Outbox},
		contacts: contactResolver{db: db, contacts: deps.Contacts, settings: deps.Settings, outbox: deps.Outbox},
	}
}

type orderToBook struct {
	order    oms.SalesOrder
	lines    []oms.SalesOrderLineItem
	payments oms.SalesOrderPayments
	customer *customersDomain.Customer
}

func (uc *SendSalesOrderUseCase) Execute(ctx context.Context, orderID string) (*mb.Document, error) {
	acc, err := loadAccount(ctx, uc.db, uc.deps.Connections, uc.deps.Settings, uc.deps.Secret)
	if err != nil {
		return nil, fmt.Errorf("sendSalesOrderToMoneybird: %w", err)
	}
	if err := acc.settings.Ready(); err != nil {
		return nil, fmt.Errorf("sendSalesOrderToMoneybird: %w", err)
	}
	src, err := uc.load(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("sendSalesOrderToMoneybird: %w", err)
	}
	prior, err := uc.recorder.find(ctx, mb.SourceSalesOrder, orderID)
	if err != nil {
		return nil, fmt.Errorf("sendSalesOrderToMoneybird: %w", err)
	}
	doc := mb.Document{
		TenantID: acc.conn.TenantID, ConnectionID: acc.conn.ID, SourceType: mb.SourceSalesOrder, SourceID: orderID,
		Reference: orderReference(src.order),
	}
	if prior != nil {
		doc.MoneybirdID, doc.PaymentRegistered = prior.MoneybirdID, prior.PaymentRegistered
	}
	var payments []mb.Payment
	if src.order.PaymentStatus() == oms.PaymentStatusPaid {
		if payments, err = paymentsFor(acc.settings, src); err != nil {
			return nil, fmt.Errorf("sendSalesOrderToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
		}
	}
	if prior != nil && prior.Status == mb.DocumentSynced && (prior.PaymentRegistered || len(payments) == 0) {
		return prior, nil
	}
	booked, err := uc.book(ctx, acc, src, &doc, payments)
	if errors.Is(err, mb.ErrDuplicateReference) {
		// An event for the same order sent at the same time may have booked
		// it a moment ago -- then the invoice is ours, not a duplicate.
		if current, findErr := uc.recorder.find(ctx, mb.SourceSalesOrder, orderID); findErr == nil && current != nil &&
			current.Status == mb.DocumentSynced && current.MoneybirdID != "" {
			return current, nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("sendSalesOrderToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
	}
	return booked, nil
}

func (uc *SendSalesOrderUseCase) load(ctx context.Context, orderID string) (*orderToBook, error) {
	var src orderToBook
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.deps.Orders.FindByID(ctx, orderID)
		if err != nil {
			return err
		}
		src.order = *o
		if src.lines, err = uc.deps.Lines.ListBySalesOrder(ctx, orderID); err != nil {
			return err
		}
		if src.payments, err = uc.deps.Payments.ListBySalesOrder(ctx, orderID); err != nil {
			return err
		}
		if o.CustomerID == nil {
			return nil
		}
		c, err := uc.deps.Customers.FindByID(ctx, *o.CustomerID)
		if errors.Is(err, customersDomain.ErrCustomerNotFound) {
			return nil // deleted since -- the order's own address still names them
		}
		src.customer = c
		return err
	})
	return &src, err
}

// book creates (or resumes) the invoice, then registers payments (none
// for an unpaid order, or a paid one without a payment method).
// doc.MoneybirdID is set as soon as the invoice exists, so a failure after
// that point keeps it for the retry.
func (uc *SendSalesOrderUseCase) book(ctx context.Context, acc *account, src *orderToBook, doc *mb.Document, payments []mb.Payment) (*mb.Document, error) {
	client := acc.client(uc.deps.Factory)

	var remote *mb.RemoteDocument
	if doc.MoneybirdID != "" {
		got, err := client.GetInvoice(ctx, doc.MoneybirdID)
		if err != nil {
			return nil, err
		}
		remote = got
	} else {
		created, err := uc.createInvoice(ctx, acc, client, src, doc.Reference)
		if err != nil {
			return nil, err
		}
		remote, doc.MoneybirdID = created, created.ID
	}

	if len(payments) > 0 {
		foreign, err := uc.foreignCurrency(ctx, acc, src.order.CurrencyCode)
		if err != nil {
			return nil, err
		}
		if err := registerPayments(ctx, client, remote, payments, foreign); err != nil {
			return nil, err
		}
		doc.PaymentRegistered = true
	}
	return uc.recorder.synced(ctx, *doc)
}

func (uc *SendSalesOrderUseCase) createInvoice(ctx context.Context, acc *account, client mb.Client, src *orderToBook, reference string) (*mb.RemoteDocument, error) {
	booked, err := mb.OrderLines(src.order, src.lines)
	if err != nil {
		return nil, err
	}
	lines, err := documentLines(acc.settings, booked)
	if err != nil {
		return nil, err
	}
	existing, err := client.FindInvoiceByReference(ctx, reference)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: %s", mb.ErrDuplicateReference, reference)
	}
	contactID, err := uc.contacts.resolve(ctx, acc, client, orderParty(src))
	if err != nil {
		return nil, err
	}
	date := src.order.OccurredAt.Format("2006-01-02")
	return client.CreateInvoice(ctx, mb.Invoice{
		ContactID: contactID, Reference: reference, Date: date, DueDate: date, Currency: src.order.CurrencyCode,
		PricesInclTax: true, Source: invoiceSource, Lines: lines,
	})
}

// paymentsFor is one payment per part of a split payment, else one for the
// whole total (see Settings.PaymentFor). None when a part has no payment
// method to register: never only part of a sale's payment.
func paymentsFor(s mb.Settings, src *orderToBook) ([]mb.Payment, error) {
	date := src.order.OccurredAt
	if src.order.DatePaid != nil {
		date = *src.order.DatePaid
	}
	day := date.Format("2006-01-02")
	parts := src.payments
	if len(parts) == 0 {
		parts = oms.SalesOrderPayments{{Method: src.order.PaymentMethod, AmountMinorUnits: src.order.TotalMinorUnits}}
	}
	out := make([]mb.Payment, 0, len(parts))
	for _, p := range parts {
		method, register, err := s.PaymentFor(p.Method)
		if err != nil {
			return nil, err
		}
		if !register {
			return nil, nil
		}
		out = append(out, mb.Payment{Date: day, Price: mb.FormatAmount(p.AmountMinorUnits), Method: method})
	}
	return out, nil
}

// foreignCurrency reports whether the order's currency differs from the
// administration's own -- Moneybird then needs the payment's base amount.
func (uc *SendSalesOrderUseCase) foreignCurrency(ctx context.Context, acc *account, currency string) (bool, error) {
	admins, err := uc.deps.Factory.New(acc.token, "").ListAdministrations(ctx)
	if err != nil {
		return false, err
	}
	for _, a := range admins {
		if a.ID == acc.settings.AdministrationID {
			return a.Currency != "" && a.Currency != currency, nil
		}
	}
	return false, nil
}

// registerPayments skips an invoice Moneybird already has as paid (e.g.
// matched to the bank meanwhile) -- never paid twice. A foreign-currency
// invoice paid in one go carries Moneybird's own base-currency total.
func registerPayments(ctx context.Context, client mb.Client, remote *mb.RemoteDocument, payments []mb.Payment, foreign bool) error {
	if remote.State == "paid" {
		return nil
	}
	if foreign && len(payments) == 1 {
		payments[0].PriceBase = remote.TotalInclTaxBase
	}
	for _, p := range payments {
		if err := client.RegisterPayment(ctx, remote.ID, p); err != nil {
			return fmt.Errorf("registering payment: %w", err)
		}
	}
	return nil
}

// orderReference is the order's own number, as the customer knows it
// (every order has one: imported orders take the shop's, migration 247).
func orderReference(o oms.SalesOrder) string { return o.Number }

func orderParty(src *orderToBook) party {
	b := src.order.Billing
	fromOrder := mb.Contact{
		CompanyName: b.Company, FirstName: b.FirstName, LastName: b.LastName, Email: b.Email,
		Address1: b.Address1, Address2: b.Address2, ZipCode: b.Postcode, City: b.City, Country: b.Country,
	}
	p := party{Contact: contactFromCustomer(src.customer, fromOrder)}
	if src.customer != nil {
		p.CustomerID = src.customer.ID
	} else if src.order.CustomerID != nil {
		p.CustomerID = *src.order.CustomerID
	}
	return p
}
