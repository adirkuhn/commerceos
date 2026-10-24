package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/quotes"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// QuoteRepo stores quotes.Quote in quotes + quote_lines, numbering them
// per tenant through quote_number_counters.
type QuoteRepo struct{ db *gorm.DB }

func NewQuoteRepo(db *gorm.DB) *QuoteRepo { return &QuoteRepo{db: db} }

var _ quotes.Repository = (*QuoteRepo)(nil)

type quoteModel struct {
	ID                  string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID            string    `gorm:"column:tenant_id"`
	Number              int64     `gorm:"column:number"`
	Name                string    `gorm:"column:name"`
	CustomerID          *string   `gorm:"column:customer_id"`
	DiscountBasisPoints int       `gorm:"column:discount_basis_points"`
	CurrencyCode        string    `gorm:"column:currency_code"`
	CreatedBy           string    `gorm:"column:created_by"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime:false"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime:false"`
}

func (quoteModel) TableName() string { return "quotes" }

type quoteLineModel struct {
	QuoteID             string `gorm:"column:quote_id;primaryKey"`
	TenantID            string `gorm:"column:tenant_id"`
	Position            int    `gorm:"column:position"`
	ProductID           string `gorm:"column:product_id"`
	SKUID               string `gorm:"column:sku_id;primaryKey"`
	Quantity            int64  `gorm:"column:quantity"`
	UnitPriceMinorUnits int64  `gorm:"column:unit_price_minor_units"`
}

func (quoteLineModel) TableName() string { return "quote_lines" }

func (r *QuoteRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("quoteRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *QuoteRepo) Create(ctx context.Context, q *quotes.Quote) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	// The counter row's lock serializes concurrent creates per tenant.
	var number int64
	if err := tx.Raw(`INSERT INTO quote_number_counters (tenant_id, last_number) VALUES (?, 1)
		ON CONFLICT (tenant_id) DO UPDATE SET last_number = quote_number_counters.last_number + 1
		RETURNING last_number`, q.TenantID).Scan(&number).Error; err != nil {
		return fmt.Errorf("quoteRepo.Create: number: %w", err)
	}
	m := toQuoteModel(q)
	m.Number = number
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("quoteRepo.Create: %w", err)
	}
	q.ID, q.Number = m.ID, number
	return r.insertLines(tx, q)
}

func (r *QuoteRepo) FindByID(ctx context.Context, id string) (*quotes.Quote, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m quoteModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, quotes.ErrQuoteNotFound
		}
		return nil, fmt.Errorf("quoteRepo.FindByID: %w", err)
	}
	var lines []quoteLineModel
	if err := tx.Where("quote_id = ?", id).Order("position").Find(&lines).Error; err != nil {
		return nil, fmt.Errorf("quoteRepo.FindByID: lines: %w", err)
	}
	q := toQuoteDomain(m)
	if q.Customer != nil {
		var c customerNameRow
		if err := tx.Raw(`SELECT first_name, last_name, billing_company, email FROM customers WHERE id = ?`, q.Customer.ID).
			Scan(&c).Error; err != nil {
			return nil, fmt.Errorf("quoteRepo.FindByID: customer: %w", err)
		}
		q.Customer.Name, q.Customer.Email = c.displayName(), c.Email
	}
	q.Lines = make([]quotes.Line, len(lines))
	for i, l := range lines {
		q.Lines[i] = quotes.Line{ProductID: l.ProductID, SKUID: l.SKUID, Quantity: l.Quantity, UnitPriceMinorUnits: l.UnitPriceMinorUnits}
	}
	return q, nil
}

// customerNameRow is the slice of customers a quote shows; named the same
// way as customersDomain.Customer.DisplayName.
type customerNameRow struct {
	FirstName      string `gorm:"column:first_name"`
	LastName       string `gorm:"column:last_name"`
	BillingCompany string `gorm:"column:billing_company"`
	Email          string `gorm:"column:email"`
}

func (c customerNameRow) displayName() string {
	return customersDomain.Customer{FirstName: c.FirstName, LastName: c.LastName, Email: c.Email,
		Billing: customersDomain.Address{Company: c.BillingCompany}}.DisplayName()
}

type quoteSummaryRow struct {
	ID                  string    `gorm:"column:id"`
	Number              int64     `gorm:"column:number"`
	Name                string    `gorm:"column:name"`
	CustomerID          *string   `gorm:"column:customer_id"`
	FirstName           string    `gorm:"column:first_name"`
	LastName            string    `gorm:"column:last_name"`
	BillingCompany      string    `gorm:"column:billing_company"`
	Email               string    `gorm:"column:email"`
	DiscountBasisPoints int       `gorm:"column:discount_basis_points"`
	CurrencyCode        string    `gorm:"column:currency_code"`
	CreatedAt           time.Time `gorm:"column:created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at"`
	LineCount           int       `gorm:"column:line_count"`
	TotalMinorUnits     int64     `gorm:"column:total_minor_units"`
}

func (r *QuoteRepo) List(ctx context.Context, filter quotes.ListFilter) ([]quotes.Summary, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	base := tx.Model(&quoteModel{})
	if filter.Search != "" {
		like := "%" + escapeLike(filter.Search) + "%"
		where := `quotes.name ILIKE ? OR quotes.customer_id IN (SELECT id FROM customers WHERE
			first_name || ' ' || last_name ILIKE ? OR billing_company ILIKE ? OR email ILIKE ?)`
		args := []any{like, like, like, like}
		if n, ok := searchedNumber(filter.Search); ok {
			where += " OR quotes.number = ?"
			args = append(args, n)
		}
		base = base.Where(where, args...)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("quoteRepo.List: count: %w", err)
	}
	var rows []quoteSummaryRow
	if err := base.Session(&gorm.Session{}).
		Select(`quotes.id, quotes.number, quotes.name, quotes.customer_id, quotes.discount_basis_points, quotes.currency_code,
			quotes.created_at, quotes.updated_at,
			COALESCE(MAX(customers.first_name), '') AS first_name, COALESCE(MAX(customers.last_name), '') AS last_name,
			COALESCE(MAX(customers.billing_company), '') AS billing_company, COALESCE(MAX(customers.email), '') AS email,
			COUNT(quote_lines.sku_id) AS line_count,
			COALESCE(SUM(quote_lines.quantity * quote_lines.unit_price_minor_units), 0) AS total_minor_units`).
		Joins("LEFT JOIN quote_lines ON quote_lines.quote_id = quotes.id").
		Joins("LEFT JOIN customers ON customers.id = quotes.customer_id").
		Group("quotes.id").
		Order("quotes.updated_at DESC, quotes.number DESC").
		Limit(filter.Limit).Offset(filter.Offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("quoteRepo.List: %w", err)
	}
	out := make([]quotes.Summary, len(rows))
	for i, row := range rows {
		out[i] = quotes.Summary{
			ID: row.ID, Number: row.Number, Name: row.Name, CurrencyCode: row.CurrencyCode,
			Customer:            summaryCustomer(row),
			DiscountBasisPoints: row.DiscountBasisPoints, LineCount: row.LineCount, TotalMinorUnits: row.TotalMinorUnits,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	return out, total, nil
}

func summaryCustomer(row quoteSummaryRow) *quotes.CustomerRef {
	if row.CustomerID == nil {
		return nil
	}
	name := customerNameRow{FirstName: row.FirstName, LastName: row.LastName, BillingCompany: row.BillingCompany, Email: row.Email}
	return &quotes.CustomerRef{ID: *row.CustomerID, Name: name.displayName(), Email: row.Email}
}

// searchedNumber reads "Q-0007", "q7" or "7" as quote number 7.
func searchedNumber(search string) (int64, bool) {
	s := strings.TrimPrefix(strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(search)), "Q"), "-")
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil && n > 0
}

func (r *QuoteRepo) Update(ctx context.Context, q *quotes.Quote) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&quoteModel{}).Where("id = ?", q.ID).Updates(map[string]any{
		"name": q.Name, "customer_id": q.CustomerID(), "discount_basis_points": q.DiscountBasisPoints,
		"currency_code": q.CurrencyCode, "updated_at": q.UpdatedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("quoteRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return quotes.ErrQuoteNotFound
	}
	if err := tx.Where("quote_id = ?", q.ID).Delete(&quoteLineModel{}).Error; err != nil {
		return fmt.Errorf("quoteRepo.Update: lines: %w", err)
	}
	return r.insertLines(tx, q)
}

func (r *QuoteRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&quoteModel{})
	if res.Error != nil {
		return fmt.Errorf("quoteRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return quotes.ErrQuoteNotFound
	}
	return nil
}

func (r *QuoteRepo) insertLines(tx *gorm.DB, q *quotes.Quote) error {
	if len(q.Lines) == 0 {
		return nil
	}
	lines := make([]quoteLineModel, len(q.Lines))
	for i, l := range q.Lines {
		lines[i] = quoteLineModel{
			QuoteID: q.ID, TenantID: q.TenantID, Position: i, ProductID: l.ProductID, SKUID: l.SKUID,
			Quantity: l.Quantity, UnitPriceMinorUnits: l.UnitPriceMinorUnits,
		}
	}
	if err := tx.Create(&lines).Error; err != nil {
		return fmt.Errorf("quoteRepo: lines: %w", err)
	}
	return nil
}

func toQuoteModel(q *quotes.Quote) quoteModel {
	return quoteModel{
		ID: q.ID, TenantID: q.TenantID, Number: q.Number, Name: q.Name, CustomerID: q.CustomerID(),
		DiscountBasisPoints: q.DiscountBasisPoints, CurrencyCode: q.CurrencyCode, CreatedBy: q.CreatedBy,
		CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
}

func toQuoteDomain(m quoteModel) *quotes.Quote {
	var customer *quotes.CustomerRef
	if m.CustomerID != nil {
		customer = &quotes.CustomerRef{ID: *m.CustomerID}
	}
	return &quotes.Quote{
		Customer: customer,
		ID:       m.ID, TenantID: m.TenantID, Number: m.Number, Name: m.Name, DiscountBasisPoints: m.DiscountBasisPoints,
		CurrencyCode: m.CurrencyCode, CreatedBy: m.CreatedBy, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
