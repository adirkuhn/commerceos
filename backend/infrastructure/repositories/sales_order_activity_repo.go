package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type salesOrderActivityModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	SalesOrderID string `gorm:"column:sales_order_id;not null;type:uuid"`
	UserID       string `gorm:"column:user_id;not null;type:uuid"`
	ActivityType string `gorm:"column:activity_type;not null"`
	Summary      string `gorm:"not null"`
	Metadata     []byte `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time
}

func (salesOrderActivityModel) TableName() string { return "sales_order_activities" }

type SalesOrderActivityRepo struct{ db *gorm.DB }

func NewSalesOrderActivityRepo(db *gorm.DB) *SalesOrderActivityRepo {
	return &SalesOrderActivityRepo{db: db}
}

func (r *SalesOrderActivityRepo) Create(ctx context.Context, a *omsDomain.SalesOrderActivity) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("salesOrderActivityRepo: operation called outside a transaction")
	}
	m := salesOrderActivityModel{
		TenantID: a.TenantID, SalesOrderID: a.SalesOrderID, UserID: a.UserID,
		ActivityType: a.ActivityType, Summary: a.Summary, Metadata: a.Metadata,
	}
	if err := tx.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("salesOrderActivityRepo.Create: %w", err)
	}
	a.ID = m.ID
	a.CreatedAt = m.CreatedAt
	return nil
}

func (r *SalesOrderActivityRepo) ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderActivity, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderActivityRepo: operation called outside a transaction")
	}
	var rows []salesOrderActivityModel
	if err := tx.WithContext(ctx).Where("sales_order_id = ?", salesOrderID).
		Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderActivityRepo.ListBySalesOrder: %w", err)
	}
	return toDomainActivities(rows), nil
}

func (r *SalesOrderActivityRepo) ListByUser(ctx context.Context, userID string) ([]omsDomain.SalesOrderActivity, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderActivityRepo: operation called outside a transaction")
	}
	var rows []salesOrderActivityModel
	if err := tx.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderActivityRepo.ListByUser: %w", err)
	}
	return toDomainActivities(rows), nil
}

func toDomainActivities(rows []salesOrderActivityModel) []omsDomain.SalesOrderActivity {
	out := make([]omsDomain.SalesOrderActivity, len(rows))
	for i, r := range rows {
		out[i] = omsDomain.SalesOrderActivity{
			ID: r.ID, TenantID: r.TenantID, SalesOrderID: r.SalesOrderID, UserID: r.UserID,
			ActivityType: r.ActivityType, Summary: r.Summary, Metadata: r.Metadata, CreatedAt: r.CreatedAt,
		}
	}
	return out
}
