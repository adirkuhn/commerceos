package moneybird

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type quoteFinder interface {
	FindByID(ctx context.Context, id string) (*quotes.Quote, error)
}

// quoteCatalog is pms' ListPriceListUseCase: each SKU's name and VAT rate,
// and whether the shop's prices include VAT.
type quoteCatalog interface {
	Execute(ctx context.Context, in apppms.ListPriceListInput) (*apppms.ListPriceListResult, error)
}

type SendQuoteDeps struct {
	Connections connectionLister
	Settings    mb.SettingsRepository
	Documents   mb.DocumentRepository
	Contacts    mb.ContactRepository
	Quotes      quoteFinder
	Catalog     quoteCatalog
	Customers   customerReader
	Factory     mb.ClientFactory
	Outbox      eventrecord.Enqueuer
	Secret      string
}

// SendQuoteUseCase sends a quote to Moneybird as an estimate: the lines
// with a quantity, at the quoted unit prices and each product's VAT rate.
// Sending again updates the same estimate while it is still a draft in
// Moneybird; once it was sent or accepted there it is locked
// (ErrDocumentLocked).
type SendQuoteUseCase struct {
	db       *gorm.DB
	deps     SendQuoteDeps
	recorder documentRecorder
	contacts contactResolver
}

func NewSendQuoteUseCase(db *gorm.DB, deps SendQuoteDeps) *SendQuoteUseCase {
	return &SendQuoteUseCase{
		db: db, deps: deps,
		recorder: documentRecorder{db: db, docs: deps.Documents, outbox: deps.Outbox},
		contacts: contactResolver{db: db, contacts: deps.Contacts, settings: deps.Settings, outbox: deps.Outbox},
	}
}

func (uc *SendQuoteUseCase) Execute(ctx context.Context, quoteID string) (*mb.Document, error) {
	acc, err := loadAccount(ctx, uc.db, uc.deps.Connections, uc.deps.Settings, uc.deps.Secret)
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", err)
	}
	if err := acc.settings.Ready(); err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", err)
	}
	q, customer, err := uc.load(ctx, quoteID)
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", err)
	}
	prior, err := uc.recorder.find(ctx, mb.SourceQuote, quoteID)
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", err)
	}
	doc := mb.Document{
		TenantID: acc.conn.TenantID, ConnectionID: acc.conn.ID, SourceType: mb.SourceQuote, SourceID: quoteID,
		Reference: quotes.Reference(q.Number),
	}
	if prior != nil {
		doc.MoneybirdID = prior.MoneybirdID
	}

	estimate, err := uc.estimate(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
	}
	estimate.Reference = doc.Reference
	client := acc.client(uc.deps.Factory)

	if doc.MoneybirdID != "" {
		current, err := client.GetEstimate(ctx, doc.MoneybirdID)
		if err != nil {
			return nil, fmt.Errorf("sendQuoteToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
		}
		if current.State != mb.EstimateDraftState {
			// The estimate itself is fine -- it just moved on in Moneybird.
			return nil, fmt.Errorf("sendQuoteToMoneybird: %w: estimate is %s", mb.ErrDocumentLocked, current.State)
		}
	}
	contactID, err := uc.contacts.resolve(ctx, acc, client, quoteParty(q, customer))
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
	}
	estimate.ContactID = contactID
	estimate.Lines, err = documentLines(acc.settings, estimate.booked)
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
	}

	var remote *mb.RemoteDocument
	if doc.MoneybirdID != "" {
		remote, err = client.UpdateEstimate(ctx, doc.MoneybirdID, estimate.Estimate)
	} else {
		remote, err = client.CreateEstimate(ctx, estimate.Estimate)
	}
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", uc.recorder.failed(ctx, doc, err))
	}
	doc.MoneybirdID = remote.ID
	synced, err := uc.recorder.synced(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("sendQuoteToMoneybird: %w", err)
	}
	return synced, nil
}

func (uc *SendQuoteUseCase) load(ctx context.Context, quoteID string) (*quotes.Quote, *customersDomain.Customer, error) {
	var q *quotes.Quote
	var customer *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if q, err = uc.deps.Quotes.FindByID(ctx, quoteID); err != nil {
			return err
		}
		if q.Customer == nil || q.Customer.ID == "" {
			return nil
		}
		customer, err = uc.deps.Customers.FindByID(ctx, q.Customer.ID)
		if errors.Is(err, customersDomain.ErrCustomerNotFound) {
			return nil
		}
		return err
	})
	return q, customer, err
}

type pendingEstimate struct {
	mb.Estimate
	booked []mb.BookedLine
}

// estimate is the quote's ordered lines with their names and VAT rates
// from the catalog.
func (uc *SendQuoteUseCase) estimate(ctx context.Context, q *quotes.Quote) (*pendingEstimate, error) {
	var ordered []quotes.Line
	var productIDs []string
	for _, l := range q.Lines {
		if l.Quantity > 0 {
			ordered = append(ordered, l)
			productIDs = append(productIDs, l.ProductID)
		}
	}
	if len(ordered) == 0 {
		return nil, mb.ErrNothingToBook
	}
	catalog, err := uc.deps.Catalog.Execute(ctx, apppms.ListPriceListInput{ProductIDs: productIDs, PerPage: quotes.MaxLines})
	if err != nil {
		return nil, err
	}
	rowBySKU := make(map[string]apppms.PriceListRow, len(catalog.Rows))
	for _, r := range catalog.Rows {
		rowBySKU[r.SKUID] = r
	}
	out := &pendingEstimate{Estimate: mb.Estimate{
		Date: q.CreatedAt.Format("2006-01-02"), Currency: q.CurrencyCode, PricesInclTax: catalog.PricesIncludeTax,
	}}
	for _, l := range ordered {
		row, ok := rowBySKU[l.SKUID]
		if !ok {
			return nil, quotes.ErrLineUnavailable
		}
		name := row.ProductName
		if row.VariantName != "" {
			name += " - " + row.VariantName
		}
		out.booked = append(out.booked, mb.QuoteLine(name, l.Quantity, l.UnitPriceMinorUnits, int32(row.TaxRateBasisPoints)))
	}
	return out, nil
}

func quoteParty(q *quotes.Quote, customer *customersDomain.Customer) party {
	var fallback mb.Contact
	p := party{}
	if q.Customer != nil {
		fallback = mb.Contact{CompanyName: q.Customer.Name, Email: q.Customer.Email}
		p.CustomerID = q.Customer.ID
	} else {
		fallback = mb.Contact{CompanyName: q.Name}
	}
	p.Contact = contactFromCustomer(customer, fallback)
	return p
}
