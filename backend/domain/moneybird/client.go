package moneybird

import "context"

// Administration is one Moneybird bookkeeping (a user can have several).
type Administration struct {
	ID       string
	Name     string
	Currency string
}

// TaxRate is a Moneybird sales tax rate. Percentage is Moneybird's decimal
// string ("21.0").
type TaxRate struct {
	ID         string
	Name       string
	Percentage string
}

type LedgerAccount struct {
	ID          string
	Name        string
	AccountType string
}

// Contact is what CommerceOS knows about a customer when it looks up or
// creates their Moneybird contact.
type Contact struct {
	ID          string
	CompanyName string
	FirstName   string
	LastName    string
	Email       string
	Address1    string
	Address2    string
	ZipCode     string
	City        string
	Country     string
}

// DocumentLine is one detail row of an estimate or invoice. Price is per
// unit, in the document's PricesInclTax convention.
type DocumentLine struct {
	Description     string
	Amount          string
	PriceMinorUnits int64
	TaxRateID       string
	LedgerAccountID string
}

// Invoice is an external sales invoice: issued by CommerceOS, booked in Moneybird.
type Invoice struct {
	ContactID     string
	Reference     string
	Date          string // YYYY-MM-DD
	DueDate       string // YYYY-MM-DD
	Currency      string
	PricesInclTax bool
	Source        string
	SourceURL     string
	Lines         []DocumentLine
}

// Estimate is a Moneybird quote.
type Estimate struct {
	ContactID     string
	Reference     string
	Date          string // YYYY-MM-DD
	Currency      string
	PricesInclTax bool
	Lines         []DocumentLine
}

// RemoteDocument is a created invoice/estimate as Moneybird reports it.
type RemoteDocument struct {
	ID    string
	State string
	// TotalUnpaid is "" for an estimate.
	TotalUnpaid string
	// TotalInclTaxBase is the total in the administration's own currency --
	// Moneybird requires it on a payment for a foreign-currency invoice.
	TotalInclTaxBase string
}

// Payment registers money received on an external sales invoice.
type Payment struct {
	Date      string // YYYY-MM-DD
	Price     string
	PriceBase string // "" unless the invoice currency differs from the administration's
	Method    string
}

// Client talks to one Moneybird administration (or, without one, to the
// account itself -- see ClientFactory).
type Client interface {
	ListAdministrations(ctx context.Context) ([]Administration, error)
	ListSalesTaxRates(ctx context.Context) ([]TaxRate, error)
	ListRevenueLedgerAccounts(ctx context.Context) ([]LedgerAccount, error)

	GetContact(ctx context.Context, id string) (*Contact, error)
	// FindContact searches by email or company name; nil when none matches.
	FindContact(ctx context.Context, query string) (*Contact, error)
	CreateContact(ctx context.Context, c Contact) (*Contact, error)

	// FindInvoiceByReference is nil when no external sales invoice has
	// exactly this reference.
	FindInvoiceByReference(ctx context.Context, reference string) (*RemoteDocument, error)
	GetInvoice(ctx context.Context, id string) (*RemoteDocument, error)
	CreateInvoice(ctx context.Context, inv Invoice) (*RemoteDocument, error)
	RegisterPayment(ctx context.Context, invoiceID string, p Payment) error

	GetEstimate(ctx context.Context, id string) (*RemoteDocument, error)
	CreateEstimate(ctx context.Context, e Estimate) (*RemoteDocument, error)
	// UpdateEstimate replaces the estimate's lines and fields; Moneybird only
	// allows it while the estimate is a draft.
	UpdateEstimate(ctx context.Context, id string, e Estimate) (*RemoteDocument, error)

	// DeleteInvoice/DeleteEstimate remove a document from Moneybird; one
	// already gone there is not an error.
	DeleteInvoice(ctx context.Context, id string) error
	DeleteEstimate(ctx context.Context, id string) error
}

// ClientFactory builds a Client for an API token, scoped to an
// administration ("" for account-level calls like ListAdministrations).
type ClientFactory interface {
	New(apiToken, administrationID string) Client
}

// EstimateDraftState is the only state in which Moneybird lets an
// estimate be changed.
const EstimateDraftState = "draft"
