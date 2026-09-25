package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type reviewModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string `gorm:"column:tenant_id;not null;type:uuid"`
	SalesChannelID string `gorm:"column:sales_channel_id;not null;type:uuid"`
	ProductID      string `gorm:"column:product_id;not null;type:uuid"`
	CustomerName   string `gorm:"column:customer_name;not null"`
	CustomerEmail  string `gorm:"column:customer_email;not null"`
	Rating         int    `gorm:"not null"`
	Title          string
	Body           string `gorm:"not null"`
	Verified       bool   `gorm:"not null;default:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (reviewModel) TableName() string { return "reviews" }

type ReviewRepo struct{ db *gorm.DB }

func NewReviewRepo(db *gorm.DB) *ReviewRepo { return &ReviewRepo{db: db} }

func (r *ReviewRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("reviewRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ReviewRepo) Create(ctx context.Context, rv *reviewsDomain.Review) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := reviewModel{
		TenantID: rv.TenantID, SalesChannelID: rv.SalesChannelID, ProductID: rv.ProductID,
		CustomerName: rv.CustomerName, CustomerEmail: rv.CustomerEmail, Rating: rv.Rating,
		Title: rv.Title, Body: rv.Body, Verified: rv.Verified, CreatedAt: rv.CreatedAt, UpdatedAt: rv.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("reviewRepo.Create: %w", err)
	}
	rv.ID = m.ID
	return nil
}

func (r *ReviewRepo) ListByProduct(ctx context.Context, filter reviewsDomain.ListFilter) ([]reviewsDomain.Review, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	var rows []reviewModel
	err = tx.Where("sales_channel_id = ? AND product_id = ?", filter.SalesChannelID, filter.ProductID).
		Order("created_at desc").Limit(limit).Offset(filter.Offset).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("reviewRepo.ListByProduct: %w", err)
	}
	out := make([]reviewsDomain.Review, len(rows))
	for i, row := range rows {
		out[i] = toReviewDomain(row)
	}
	return out, nil
}

func (r *ReviewRepo) SummaryByProduct(ctx context.Context, salesChannelID, productID string) (reviewsDomain.Summary, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return reviewsDomain.Summary{}, err
	}
	var row struct {
		Average float64
		Count   int
	}
	err = tx.Model(&reviewModel{}).
		Select("COALESCE(AVG(rating), 0) AS average, COUNT(*) AS count").
		Where("sales_channel_id = ? AND product_id = ?", salesChannelID, productID).
		Scan(&row).Error
	if err != nil {
		return reviewsDomain.Summary{}, fmt.Errorf("reviewRepo.SummaryByProduct: %w", err)
	}
	return reviewsDomain.Summary{Average: row.Average, Count: row.Count}, nil
}

func (r *ReviewRepo) ListAll(ctx context.Context) ([]reviewsDomain.Review, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []reviewModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("reviewRepo.ListAll: %w", err)
	}
	out := make([]reviewsDomain.Review, len(rows))
	for i, row := range rows {
		out[i] = toReviewDomain(row)
	}
	return out, nil
}

func (r *ReviewRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&reviewModel{}).Error; err != nil {
		return fmt.Errorf("reviewRepo.Delete: %w", err)
	}
	return nil
}

// HasVerifiedPurchase joins sales_order_line_items -> sales_orders and
// sales_order_line_items -> skus -> variants to answer "did this email
// complete an order on this channel containing this product" -- a raw
// cross-domain query kept in infrastructure (not a domain/oms or domain/pms
// repository method) since RLS on every joined table already scopes it to
// the current tenant, same as any other query run inside tenant.WithTenant.
func (r *ReviewRepo) HasVerifiedPurchase(ctx context.Context, salesChannelID, productID, email string) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = tx.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM sales_order_line_items li
			JOIN sales_orders so ON so.id = li.sales_order_id
			JOIN skus sk ON sk.id = li.sku_id
			JOIN variants v ON v.id = sk.variant_id
			WHERE so.sales_channel_id = ?
			  AND so.status = ?
			  AND LOWER(so.billing_email) = LOWER(?)
			  AND v.product_id = ?
		)`, salesChannelID, string(omsDomain.SalesOrderStatusCompleted), email, productID).Scan(&exists).Error
	if err != nil {
		return false, fmt.Errorf("reviewRepo.HasVerifiedPurchase: %w", err)
	}
	return exists, nil
}

func toReviewDomain(m reviewModel) reviewsDomain.Review {
	return reviewsDomain.Review{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, ProductID: m.ProductID,
		CustomerName: m.CustomerName, CustomerEmail: m.CustomerEmail, Rating: m.Rating,
		Title: m.Title, Body: m.Body, Verified: m.Verified, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
