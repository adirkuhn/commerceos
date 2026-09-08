package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type skuCurrencyPriceModel struct {
	ID                  string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID            string `gorm:"not null;type:uuid"`
	SKUID               string `gorm:"column:sku_id;not null;type:uuid"`
	CurrencyCode        string `gorm:"not null"`
	PriceMinorUnits     int64  `gorm:"not null"`
	SalePriceMinorUnits *int64
	SaleEndDate         *time.Time `gorm:"type:date"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (skuCurrencyPriceModel) TableName() string { return "sku_currency_prices" }

type SKUCurrencyPriceRepo struct{ db *gorm.DB }

func NewSKUCurrencyPriceRepo(db *gorm.DB) *SKUCurrencyPriceRepo { return &SKUCurrencyPriceRepo{db: db} }

func (r *SKUCurrencyPriceRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SKUCurrencyPriceRepo) FindBySKUAndCurrency(ctx context.Context, skuID, currencyCode string) (*pmsDomain.SKUCurrencyPrice, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m skuCurrencyPriceModel
	err = tx.Where("sku_id = ? AND currency_code = ?", skuID, currencyCode).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("skuCurrencyPriceRepo.FindBySKUAndCurrency: %w", err)
	}
	p := toSKUCurrencyPriceDomain(m)
	return &p, nil
}

func (r *SKUCurrencyPriceRepo) ListBySKU(ctx context.Context, skuID string) ([]pmsDomain.SKUCurrencyPrice, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var models []skuCurrencyPriceModel
	if err := tx.Where("sku_id = ?", skuID).Order("currency_code ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("skuCurrencyPriceRepo.ListBySKU: %w", err)
	}
	out := make([]pmsDomain.SKUCurrencyPrice, 0, len(models))
	for _, m := range models {
		out = append(out, toSKUCurrencyPriceDomain(m))
	}
	return out, nil
}

func (r *SKUCurrencyPriceRepo) Upsert(ctx context.Context, p *pmsDomain.SKUCurrencyPrice) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := skuCurrencyPriceModel{
		TenantID: p.TenantID, SKUID: p.SKUID, CurrencyCode: p.CurrencyCode,
		PriceMinorUnits: p.PriceMinorUnits, SalePriceMinorUnits: p.SalePriceMinorUnits, SaleEndDate: p.SaleEndDate,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "sku_id"}, {Name: "currency_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"price_minor_units", "sale_price_minor_units", "sale_end_date", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("skuCurrencyPriceRepo.Upsert: %w", err)
	}
	p.ID = m.ID
	return nil
}

func (r *SKUCurrencyPriceRepo) Delete(ctx context.Context, skuID, currencyCode string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("sku_id = ? AND currency_code = ?", skuID, currencyCode).Delete(&skuCurrencyPriceModel{}).Error; err != nil {
		return fmt.Errorf("skuCurrencyPriceRepo.Delete: %w", err)
	}
	return nil
}

func toSKUCurrencyPriceDomain(m skuCurrencyPriceModel) pmsDomain.SKUCurrencyPrice {
	return pmsDomain.SKUCurrencyPrice{
		ID: m.ID, TenantID: m.TenantID, SKUID: m.SKUID, CurrencyCode: m.CurrencyCode,
		PriceMinorUnits: m.PriceMinorUnits, SalePriceMinorUnits: m.SalePriceMinorUnits, SaleEndDate: m.SaleEndDate,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
