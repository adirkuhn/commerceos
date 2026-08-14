package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type taxClassModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	Name      string `gorm:"not null"`
	IsDefault bool   `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (taxClassModel) TableName() string { return "tax_classes" }

type taxRateModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	TaxClassID      string `gorm:"not null;type:uuid"`
	Country         string `gorm:"not null"`
	RateBasisPoints int32  `gorm:"not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (taxRateModel) TableName() string { return "tax_rates" }

// ── TaxClassRepo ──────────────────────────────────────────────────────────

type TaxClassRepo struct{ db *gorm.DB }

func NewTaxClassRepo(db *gorm.DB) *TaxClassRepo { return &TaxClassRepo{db: db} }

func (r *TaxClassRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("tax: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *TaxClassRepo) Create(ctx context.Context, c *taxDomain.TaxClass) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := taxClassModel{
		TenantID: c.TenantID, Name: c.Name, IsDefault: c.IsDefault,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("taxClassRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *TaxClassRepo) FindByID(ctx context.Context, id string) (*taxDomain.TaxClass, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m taxClassModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, taxDomain.ErrTaxClassNotFound
		}
		return nil, fmt.Errorf("taxClassRepo.FindByID: %w", err)
	}
	return toTaxClassDomain(m), nil
}

func (r *TaxClassRepo) List(ctx context.Context) ([]taxDomain.TaxClass, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []taxClassModel
	if err := tx.Order("name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("taxClassRepo.List: %w", err)
	}
	classes := make([]taxDomain.TaxClass, len(rows))
	for i, row := range rows {
		classes[i] = *toTaxClassDomain(row)
	}
	return classes, nil
}

func (r *TaxClassRepo) Update(ctx context.Context, c *taxDomain.TaxClass) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&taxClassModel{}).Where("id = ?", c.ID).
		Updates(map[string]any{
			"name":       c.Name,
			"is_default": c.IsDefault,
			"updated_at": c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("taxClassRepo.Update: %w", err)
	}
	return nil
}

func (r *TaxClassRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&taxClassModel{}).Error; err != nil {
		return fmt.Errorf("taxClassRepo.Delete: %w", err)
	}
	return nil
}

func toTaxClassDomain(m taxClassModel) *taxDomain.TaxClass {
	return &taxDomain.TaxClass{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, IsDefault: m.IsDefault,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// ── TaxRateRepo ───────────────────────────────────────────────────────────

type TaxRateRepo struct{ db *gorm.DB }

func NewTaxRateRepo(db *gorm.DB) *TaxRateRepo { return &TaxRateRepo{db: db} }

func (r *TaxRateRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("tax: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert inserts or updates by the (tenant_id, tax_class_id, country)
// unique constraint (see migration 058) -- ON CONFLICT so a repeat save of
// the same class+country pair overwrites the rate instead of erroring.
func (r *TaxRateRepo) Upsert(ctx context.Context, rate *taxDomain.TaxRate) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := taxRateModel{
		TenantID: rate.TenantID, TaxClassID: rate.TaxClassID, Country: rate.Country,
		RateBasisPoints: rate.RateBasisPoints, CreatedAt: rate.CreatedAt, UpdatedAt: rate.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "tax_class_id"}, {Name: "country"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate_basis_points", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("taxRateRepo.Upsert: %w", err)
	}
	rate.ID = m.ID
	return nil
}

func (r *TaxRateRepo) FindByID(ctx context.Context, id string) (*taxDomain.TaxRate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m taxRateModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, taxDomain.ErrTaxRateNotFound
		}
		return nil, fmt.Errorf("taxRateRepo.FindByID: %w", err)
	}
	return toTaxRateDomain(m), nil
}

func (r *TaxRateRepo) List(ctx context.Context) ([]taxDomain.TaxRate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []taxRateModel
	if err := tx.Order("country asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("taxRateRepo.List: %w", err)
	}
	rates := make([]taxDomain.TaxRate, len(rows))
	for i, row := range rows {
		rates[i] = *toTaxRateDomain(row)
	}
	return rates, nil
}

func (r *TaxRateRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&taxRateModel{}).Error; err != nil {
		return fmt.Errorf("taxRateRepo.Delete: %w", err)
	}
	return nil
}

func toTaxRateDomain(m taxRateModel) *taxDomain.TaxRate {
	return &taxDomain.TaxRate{
		ID: m.ID, TenantID: m.TenantID, TaxClassID: m.TaxClassID, Country: m.Country,
		RateBasisPoints: m.RateBasisPoints, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// ── ExternalTaxClassLinkRepo ─────────────────────────────────────────────

type externalTaxClassLinkModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	Provider     string `gorm:"not null"`
	ExternalSlug string `gorm:"not null"`
	TaxClassID   string `gorm:"not null;type:uuid"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (externalTaxClassLinkModel) TableName() string { return "external_tax_class_links" }

type ExternalTaxClassLinkRepo struct{ db *gorm.DB }

func NewExternalTaxClassLinkRepo(db *gorm.DB) *ExternalTaxClassLinkRepo {
	return &ExternalTaxClassLinkRepo{db: db}
}

func (r *ExternalTaxClassLinkRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("tax: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert inserts or updates by the (tenant_id, provider, external_slug)
// unique constraint (see migration 060).
func (r *ExternalTaxClassLinkRepo) Upsert(ctx context.Context, l *taxDomain.ExternalTaxClassLink) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := externalTaxClassLinkModel{
		TenantID: l.TenantID, Provider: l.Provider, ExternalSlug: l.ExternalSlug,
		TaxClassID: l.TaxClassID, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "provider"}, {Name: "external_slug"}},
		DoUpdates: clause.AssignmentColumns([]string{"tax_class_id", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("externalTaxClassLinkRepo.Upsert: %w", err)
	}
	l.ID = m.ID
	return nil
}

func (r *ExternalTaxClassLinkRepo) FindByProviderAndSlug(ctx context.Context, provider, externalSlug string) (*taxDomain.ExternalTaxClassLink, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m externalTaxClassLinkModel
	if err := tx.Where("provider = ? AND external_slug = ?", provider, externalSlug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, taxDomain.ErrExternalTaxClassLinkNotFound
		}
		return nil, fmt.Errorf("externalTaxClassLinkRepo.FindByProviderAndSlug: %w", err)
	}
	return toExternalTaxClassLinkDomain(m), nil
}

func (r *ExternalTaxClassLinkRepo) List(ctx context.Context) ([]taxDomain.ExternalTaxClassLink, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []externalTaxClassLinkModel
	if err := tx.Order("provider asc, external_slug asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("externalTaxClassLinkRepo.List: %w", err)
	}
	links := make([]taxDomain.ExternalTaxClassLink, len(rows))
	for i, row := range rows {
		links[i] = *toExternalTaxClassLinkDomain(row)
	}
	return links, nil
}

func (r *ExternalTaxClassLinkRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&externalTaxClassLinkModel{}).Error; err != nil {
		return fmt.Errorf("externalTaxClassLinkRepo.Delete: %w", err)
	}
	return nil
}

func toExternalTaxClassLinkDomain(m externalTaxClassLinkModel) *taxDomain.ExternalTaxClassLink {
	return &taxDomain.ExternalTaxClassLink{
		ID: m.ID, TenantID: m.TenantID, Provider: m.Provider, ExternalSlug: m.ExternalSlug,
		TaxClassID: m.TaxClassID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
