package moneybird

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

type contactJSON struct {
	ID                   string `json:"id,omitempty"`
	CompanyName          string `json:"company_name,omitempty"`
	FirstName            string `json:"firstname,omitempty"`
	LastName             string `json:"lastname,omitempty"`
	Email                string `json:"email,omitempty"`
	SendInvoicesToEmail  string `json:"send_invoices_to_email,omitempty"`
	SendEstimatesToEmail string `json:"send_estimates_to_email,omitempty"`
	Address1             string `json:"address1,omitempty"`
	Address2             string `json:"address2,omitempty"`
	ZipCode              string `json:"zipcode,omitempty"`
	City                 string `json:"city,omitempty"`
	Country              string `json:"country,omitempty"`
}

func (j contactJSON) toDomain() *mb.Contact {
	email := j.SendInvoicesToEmail
	if email == "" {
		email = j.Email
	}
	return &mb.Contact{
		ID: j.ID, CompanyName: j.CompanyName, FirstName: j.FirstName, LastName: j.LastName, Email: email,
		Address1: j.Address1, Address2: j.Address2, ZipCode: j.ZipCode, City: j.City, Country: j.Country,
	}
}

func (c *client) GetContact(ctx context.Context, id string) (*mb.Contact, error) {
	var j contactJSON
	if err := c.do(ctx, http.MethodGet, "/contacts/"+url.PathEscape(id), nil, nil, &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

// FindContact: Moneybird's query search is fuzzy, so only a contact whose
// email or company name is exactly the query counts.
func (c *client) FindContact(ctx context.Context, query string) (*mb.Contact, error) {
	var rows []contactJSON
	if err := c.do(ctx, http.MethodGet, "/contacts", url.Values{"query": {query}}, nil, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if strings.EqualFold(r.Email, query) || strings.EqualFold(r.SendInvoicesToEmail, query) || strings.EqualFold(r.CompanyName, query) {
			return r.toDomain(), nil
		}
	}
	return nil, nil
}

func (c *client) CreateContact(ctx context.Context, in mb.Contact) (*mb.Contact, error) {
	body := map[string]contactJSON{"contact": {
		CompanyName: in.CompanyName, FirstName: in.FirstName, LastName: in.LastName,
		SendInvoicesToEmail: in.Email, SendEstimatesToEmail: in.Email,
		Address1: in.Address1, Address2: in.Address2, ZipCode: in.ZipCode, City: in.City, Country: in.Country,
	}}
	var j contactJSON
	if err := c.do(ctx, http.MethodPost, "/contacts", nil, body, &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

type remoteJSON struct {
	ID                    string `json:"id"`
	Reference             string `json:"reference"`
	State                 string `json:"state"`
	TotalUnpaid           string `json:"total_unpaid"`
	TotalPriceInclTaxBase string `json:"total_price_incl_tax_base"`
	Details               []struct {
		ID string `json:"id"`
	} `json:"details"`
}

func (j remoteJSON) toDomain() *mb.RemoteDocument {
	return &mb.RemoteDocument{ID: j.ID, State: j.State, TotalUnpaid: j.TotalUnpaid, TotalInclTaxBase: j.TotalPriceInclTaxBase}
}

type lineJSON struct {
	ID              string `json:"id,omitempty"`
	Destroy         bool   `json:"_destroy,omitempty"`
	Description     string `json:"description,omitempty"`
	Amount          string `json:"amount,omitempty"`
	Price           string `json:"price,omitempty"`
	TaxRateID       string `json:"tax_rate_id,omitempty"`
	LedgerAccountID string `json:"ledger_account_id,omitempty"`
}

func linesJSON(lines []mb.DocumentLine) []lineJSON {
	out := make([]lineJSON, len(lines))
	for i, l := range lines {
		out[i] = lineJSON{
			Description: l.Description, Amount: l.Amount, Price: mb.FormatAmount(l.PriceMinorUnits),
			TaxRateID: l.TaxRateID, LedgerAccountID: l.LedgerAccountID,
		}
	}
	return out
}

func (c *client) FindInvoiceByReference(ctx context.Context, reference string) (*mb.RemoteDocument, error) {
	var rows []remoteJSON
	q := url.Values{"filter": {"reference:" + reference}}
	if err := c.do(ctx, http.MethodGet, "/external_sales_invoices", q, nil, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.Reference == reference {
			return r.toDomain(), nil
		}
	}
	return nil, nil
}

func (c *client) GetInvoice(ctx context.Context, id string) (*mb.RemoteDocument, error) {
	var j remoteJSON
	if err := c.do(ctx, http.MethodGet, "/external_sales_invoices/"+url.PathEscape(id), nil, nil, &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

func (c *client) CreateInvoice(ctx context.Context, inv mb.Invoice) (*mb.RemoteDocument, error) {
	body := map[string]any{"external_sales_invoice": map[string]any{
		"contact_id": inv.ContactID, "reference": inv.Reference, "date": inv.Date, "due_date": inv.DueDate,
		"currency": inv.Currency, "prices_are_incl_tax": inv.PricesInclTax,
		"source": inv.Source, "source_url": inv.SourceURL, "details_attributes": linesJSON(inv.Lines),
	}}
	var j remoteJSON
	if err := c.do(ctx, http.MethodPost, "/external_sales_invoices", nil, body, &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

func (c *client) RegisterPayment(ctx context.Context, invoiceID string, p mb.Payment) error {
	payment := map[string]string{"payment_date": p.Date, "price": p.Price, "payment_method": p.Method}
	if p.PriceBase != "" {
		payment["price_base"] = p.PriceBase
	}
	return c.do(ctx, http.MethodPost, "/external_sales_invoices/"+url.PathEscape(invoiceID)+"/payments", nil,
		map[string]any{"payment": payment}, nil)
}

func (c *client) GetEstimate(ctx context.Context, id string) (*mb.RemoteDocument, error) {
	var j remoteJSON
	if err := c.do(ctx, http.MethodGet, "/estimates/"+url.PathEscape(id), nil, nil, &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

func estimateBody(e mb.Estimate, details []lineJSON) map[string]any {
	return map[string]any{"estimate": map[string]any{
		"contact_id": e.ContactID, "reference": e.Reference, "estimate_date": e.Date, "currency": e.Currency,
		"prices_are_incl_tax": e.PricesInclTax, "details_attributes": details,
	}}
}

func (c *client) CreateEstimate(ctx context.Context, e mb.Estimate) (*mb.RemoteDocument, error) {
	var j remoteJSON
	if err := c.do(ctx, http.MethodPost, "/estimates", nil, estimateBody(e, linesJSON(e.Lines)), &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

// UpdateEstimate: Moneybird keeps detail rows until told to destroy them by
// id, so the current rows are read first and removed alongside the new ones.
func (c *client) UpdateEstimate(ctx context.Context, id string, e mb.Estimate) (*mb.RemoteDocument, error) {
	var current remoteJSON
	path := "/estimates/" + url.PathEscape(id)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &current); err != nil {
		return nil, err
	}
	details := make([]lineJSON, 0, len(current.Details)+len(e.Lines))
	for _, d := range current.Details {
		details = append(details, lineJSON{ID: d.ID, Destroy: true})
	}
	details = append(details, linesJSON(e.Lines)...)
	var j remoteJSON
	if err := c.do(ctx, http.MethodPatch, path, nil, estimateBody(e, details), &j); err != nil {
		return nil, err
	}
	return j.toDomain(), nil
}

func (c *client) DeleteInvoice(ctx context.Context, id string) error {
	return c.deleteGone(ctx, "/external_sales_invoices/"+url.PathEscape(id))
}

func (c *client) DeleteEstimate(ctx context.Context, id string) error {
	return c.deleteGone(ctx, "/estimates/"+url.PathEscape(id))
}

// deleteGone deletes; a document already gone in Moneybird is not an error.
func (c *client) deleteGone(ctx context.Context, path string) error {
	err := c.do(ctx, http.MethodDelete, path, nil, nil, nil)
	if errors.Is(err, errNotFound) {
		return nil
	}
	return err
}
