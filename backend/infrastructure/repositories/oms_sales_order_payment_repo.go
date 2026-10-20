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

type salesOrderPaymentModel struct {
	ID               string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID         string `gorm:"not null;type:uuid"`
	SalesOrderID     string `gorm:"column:sales_order_id;not null;type:uuid"`
	Position         int    `gorm:"not null"`
	Method           string `gorm:"not null"`
	AmountMinorUnits int64  `gorm:"not null"`
	TransactionID    string `gorm:"column:transaction_id;not null;default:''"`
	CreatedAt        time.Time
}

func (salesOrderPaymentModel) TableName() string { return "sales_order_payments" }

// SalesOrderPaymentRepo implements omsDomain.SalesOrderPaymentRepository.
type SalesOrderPaymentRepo struct{ db *gorm.DB }

func NewSalesOrderPaymentRepo(db *gorm.DB) *SalesOrderPaymentRepo {
	return &SalesOrderPaymentRepo{db: db}
}

func (r *SalesOrderPaymentRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderPaymentRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SalesOrderPaymentRepo) CreateForSalesOrder(ctx context.Context, o *omsDomain.SalesOrder) error {
	if len(o.Payments) == 0 {
		return nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	rows := make([]salesOrderPaymentModel, len(o.Payments))
	for i, p := range o.Payments {
		rows[i] = salesOrderPaymentModel{
			TenantID: o.TenantID, SalesOrderID: o.ID, Position: i,
			Method: p.Method, AmountMinorUnits: p.AmountMinorUnits, TransactionID: p.TransactionID,
		}
	}
	if err := tx.Create(&rows).Error; err != nil {
		return fmt.Errorf("salesOrderPaymentRepo.CreateForSalesOrder: %w", err)
	}
	return nil
}

func (r *SalesOrderPaymentRepo) ListBySalesOrder(ctx context.Context, salesOrderID string) (omsDomain.SalesOrderPayments, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesOrderPaymentModel
	if err := tx.Where("sales_order_id = ?", salesOrderID).Order("position asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderPaymentRepo.ListBySalesOrder: %w", err)
	}
	payments := make(omsDomain.SalesOrderPayments, len(rows))
	for i, m := range rows {
		payments[i] = omsDomain.SalesOrderPayment{Method: m.Method, AmountMinorUnits: m.AmountMinorUnits, TransactionID: m.TransactionID}
	}
	return payments, nil
}
