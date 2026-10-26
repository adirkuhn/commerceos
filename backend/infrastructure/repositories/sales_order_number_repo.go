package repositories

import (
	"context"
	"errors"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
	"gorm.io/gorm"
)

// firstOrderNumber is the first number a tenant's counter hands out.
const firstOrderNumber = 1001

// SalesOrderNumberRepo hands out order numbers per tenant through
// sales_order_number_counters.
type SalesOrderNumberRepo struct {
	db *gorm.DB
}

func NewSalesOrderNumberRepo(db *gorm.DB) *SalesOrderNumberRepo {
	return &SalesOrderNumberRepo{db: db}
}

func (r *SalesOrderNumberRepo) Next(ctx context.Context) (string, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return "", errors.New("salesOrderNumberRepo: operation called outside a transaction")
	}
	// The counter row's lock serializes concurrent creates per tenant.
	var n int64
	if err := tx.WithContext(ctx).Raw(`INSERT INTO sales_order_number_counters (tenant_id, last_number) VALUES (?, ?)
		ON CONFLICT (tenant_id) DO UPDATE SET last_number = sales_order_number_counters.last_number + 1
		RETURNING last_number`, tenant.IDFromContext(ctx), firstOrderNumber).Scan(&n).Error; err != nil {
		return "", fmt.Errorf("salesOrderNumberRepo.Next: %w", err)
	}
	return omsDomain.FormatOrderNumber(n), nil
}
