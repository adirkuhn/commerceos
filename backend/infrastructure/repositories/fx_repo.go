package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type exchangeRateModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	CurrencyCode    string `gorm:"not null"`
	RateBasisPoints int64  `gorm:"not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (exchangeRateModel) TableName() string { return "fx_rates" }

type ExchangeRateRepo struct{ db *gorm.DB }

func NewExchangeRateRepo(db *gorm.DB) *ExchangeRateRepo { return &ExchangeRateRepo{db: db} }

func (r *ExchangeRateRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("fx: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ExchangeRateRepo) FindByCurrency(ctx context.Context, currencyCode string) (*fxDomain.ExchangeRate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m exchangeRateModel
	err = tx.Where("currency_code = ?", currencyCode).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("exchangeRateRepo.FindByCurrency: %w", err)
	}
	rate := toExchangeRateDomain(m)
	return &rate, nil
}

func (r *ExchangeRateRepo) ListByTenant(ctx context.Context) ([]fxDomain.ExchangeRate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var models []exchangeRateModel
	if err := tx.Order("currency_code ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("exchangeRateRepo.ListByTenant: %w", err)
	}
	rates := make([]fxDomain.ExchangeRate, 0, len(models))
	for _, m := range models {
		rates = append(rates, toExchangeRateDomain(m))
	}
	return rates, nil
}

func (r *ExchangeRateRepo) Upsert(ctx context.Context, rate *fxDomain.ExchangeRate) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := exchangeRateModel{
		TenantID: rate.TenantID, CurrencyCode: rate.CurrencyCode, RateBasisPoints: rate.RateBasisPoints,
		CreatedAt: rate.CreatedAt, UpdatedAt: rate.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "currency_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate_basis_points", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("exchangeRateRepo.Upsert: %w", err)
	}
	rate.ID = m.ID
	return nil
}

func (r *ExchangeRateRepo) Delete(ctx context.Context, currencyCode string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("currency_code = ?", currencyCode).Delete(&exchangeRateModel{}).Error; err != nil {
		return fmt.Errorf("exchangeRateRepo.Delete: %w", err)
	}
	return nil
}

func toExchangeRateDomain(m exchangeRateModel) fxDomain.ExchangeRate {
	return fxDomain.ExchangeRate{
		ID: m.ID, TenantID: m.TenantID, CurrencyCode: m.CurrencyCode, RateBasisPoints: m.RateBasisPoints,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
