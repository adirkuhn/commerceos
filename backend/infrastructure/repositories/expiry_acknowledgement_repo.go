package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ExpiryAcknowledgementRepo stores invDomain.ExpiryAcknowledgement rows.
type ExpiryAcknowledgementRepo struct{ db *gorm.DB }

func NewExpiryAcknowledgementRepo(db *gorm.DB) *ExpiryAcknowledgementRepo {
	return &ExpiryAcknowledgementRepo{db: db}
}

var _ invDomain.ExpiryAcknowledgementRepository = (*ExpiryAcknowledgementRepo)(nil)

type expiryAcknowledgementModel struct {
	TenantID       string    `gorm:"column:tenant_id"`
	BatchID        string    `gorm:"column:batch_id;primaryKey"`
	WindowDays     int       `gorm:"column:window_days;primaryKey"`
	AcknowledgedBy string    `gorm:"column:acknowledged_by"`
	AcknowledgedAt time.Time `gorm:"column:acknowledged_at"`
}

func (expiryAcknowledgementModel) TableName() string { return "batch_expiry_acknowledgements" }

func (r *ExpiryAcknowledgementRepo) Create(ctx context.Context, ack invDomain.ExpiryAcknowledgement) (bool, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return false, errors.New("expiryAcknowledgementRepo: operation called outside a transaction")
	}
	m := expiryAcknowledgementModel{
		TenantID: ack.TenantID, BatchID: ack.BatchID, WindowDays: ack.WindowDays,
		AcknowledgedBy: ack.AcknowledgedBy, AcknowledgedAt: ack.AcknowledgedAt,
	}
	res := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&m)
	if res.Error != nil {
		return false, fmt.Errorf("expiryAcknowledgementRepo.Create: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (r *ExpiryAcknowledgementRepo) Delete(ctx context.Context, batchID string, windowDays int) (bool, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return false, errors.New("expiryAcknowledgementRepo: operation called outside a transaction")
	}
	res := tx.WithContext(ctx).Where("batch_id = ? AND window_days = ?", batchID, windowDays).Delete(&expiryAcknowledgementModel{})
	if res.Error != nil {
		return false, fmt.Errorf("expiryAcknowledgementRepo.Delete: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}
