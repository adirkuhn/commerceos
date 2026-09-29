package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── Sales channel repo ───────────────────────────────────────────────────────

// salesChannelQuantityBreaks is JSONB-backed, same driver.Valuer/
// sql.Scanner + whole-document-replace convention as theme_repo.go's
// themeBlocks type.
type salesChannelQuantityBreaks []channelsDomain.QuantityBreak

func (b salesChannelQuantityBreaks) Value() (driver.Value, error) {
	if len(b) == 0 {
		return "[]", nil
	}
	out, err := json.Marshal([]channelsDomain.QuantityBreak(b))
	return string(out), err
}

func (b *salesChannelQuantityBreaks) Scan(value any) error {
	if value == nil {
		*b = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("salesChannelQuantityBreaks.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(raw, (*[]channelsDomain.QuantityBreak)(b))
}

type salesChannelModel struct {
	ID                         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                   string `gorm:"not null;type:uuid"`
	Name                       string `gorm:"not null"`
	Type                       string `gorm:"not null"`
	ConnectionID               *string
	EnrollmentMode             string `gorm:"column:enrollment_mode;not null"`
	EnrollmentCategoryID       *string
	EnrollmentTag              string `gorm:"not null;default:''"`
	PaymentConnectionID        *string
	PriceAdjustmentBasisPoints *int                       `gorm:"column:price_adjustment_basis_points"`
	QuantityBreaks             salesChannelQuantityBreaks `gorm:"column:quantity_breaks;type:jsonb;not null;default:'[]'"`
	CurrencyCode               string                     `gorm:"column:currency_code;not null;default:''"`
	DefaultWarehouseID         *string                    `gorm:"column:default_warehouse_id;type:uuid"`
	StorefrontMode             string                     `gorm:"column:storefront_mode;not null;default:'headless'"`
	SignInURL                  *string                    `gorm:"column:sign_in_url"`
	StorefrontLocale           string                     `gorm:"column:storefront_locale;not null;default:'en'"`
	Active                     bool                       `gorm:"not null;default:true"`
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

func (salesChannelModel) TableName() string { return "sales_channels" }

type SalesChannelRepo struct{ db *gorm.DB }

func NewSalesChannelRepo(db *gorm.DB) *SalesChannelRepo { return &SalesChannelRepo{db: db} }

func (r *SalesChannelRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SalesChannelRepo) Create(ctx context.Context, c *channelsDomain.SalesChannel) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := salesChannelModel{
		TenantID: c.TenantID, Name: c.Name, Type: string(c.Type), ConnectionID: c.ConnectionID,
		EnrollmentMode: string(c.EnrollmentMode), EnrollmentCategoryID: c.EnrollmentCategoryID, EnrollmentTag: c.EnrollmentTag,
		PaymentConnectionID:        c.PaymentConnectionID,
		PriceAdjustmentBasisPoints: c.PriceAdjustmentBasisPoints,
		QuantityBreaks:             salesChannelQuantityBreaks(c.QuantityBreaks),
		CurrencyCode:               c.CurrencyCode,
		DefaultWarehouseID:         c.DefaultWarehouseID,
		StorefrontMode:             storefrontModeOrHeadless(c.StorefrontMode),
		SignInURL:                  c.SignInURL,
		StorefrontLocale:           storefrontLocaleOrEnglish(c.StorefrontLocale),
		Active:                     c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("salesChannelRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *SalesChannelRepo) FindByID(ctx context.Context, id string) (*channelsDomain.SalesChannel, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesChannelModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrSalesChannelNotFound
		}
		return nil, fmt.Errorf("salesChannelRepo.FindByID: %w", err)
	}
	return toSalesChannelDomain(m), nil
}

func (r *SalesChannelRepo) ListByTenant(ctx context.Context) ([]channelsDomain.SalesChannel, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesChannelModel
	if err := tx.Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesChannelRepo.ListByTenant: %w", err)
	}
	out := make([]channelsDomain.SalesChannel, len(rows))
	for i, row := range rows {
		out[i] = *toSalesChannelDomain(row)
	}
	return out, nil
}

func (r *SalesChannelRepo) Update(ctx context.Context, c *channelsDomain.SalesChannel) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&salesChannelModel{}).Where("id = ?", c.ID).
		Updates(map[string]any{
			"name": c.Name, "connection_id": c.ConnectionID,
			"enrollment_mode": string(c.EnrollmentMode), "enrollment_category_id": c.EnrollmentCategoryID, "enrollment_tag": c.EnrollmentTag,
			"payment_connection_id":         c.PaymentConnectionID,
			"price_adjustment_basis_points": c.PriceAdjustmentBasisPoints,
			"quantity_breaks":               salesChannelQuantityBreaks(c.QuantityBreaks),
			"default_warehouse_id":          c.DefaultWarehouseID,
			"storefront_mode":               storefrontModeOrHeadless(c.StorefrontMode),
			"sign_in_url":                   c.SignInURL,
			"storefront_locale":             storefrontLocaleOrEnglish(c.StorefrontLocale),
			"active":                        c.Active, "updated_at": c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("salesChannelRepo.Update: %w", err)
	}
	return nil
}

func (r *SalesChannelRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&salesChannelModel{}).Error; err != nil {
		return fmt.Errorf("salesChannelRepo.Delete: %w", err)
	}
	return nil
}

func toSalesChannelDomain(m salesChannelModel) *channelsDomain.SalesChannel {
	return &channelsDomain.SalesChannel{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, Type: channelsDomain.ChannelType(m.Type), ConnectionID: m.ConnectionID,
		EnrollmentMode: channelsDomain.EnrollmentMode(m.EnrollmentMode), EnrollmentCategoryID: m.EnrollmentCategoryID, EnrollmentTag: m.EnrollmentTag,
		PaymentConnectionID:        m.PaymentConnectionID,
		PriceAdjustmentBasisPoints: m.PriceAdjustmentBasisPoints,
		QuantityBreaks:             []channelsDomain.QuantityBreak(m.QuantityBreaks),
		CurrencyCode:               m.CurrencyCode,
		DefaultWarehouseID:         m.DefaultWarehouseID,
		StorefrontMode:             channelsDomain.StorefrontMode(m.StorefrontMode),
		SignInURL:                  m.SignInURL,
		StorefrontLocale:           channelsDomain.StorefrontLocale(m.StorefrontLocale),
		Active:                     m.Active, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func storefrontModeOrHeadless(m channelsDomain.StorefrontMode) string {
	if m == "" {
		return string(channelsDomain.StorefrontModeHeadless)
	}
	return string(m)
}

// ── Sales channel product override repo ─────────────────────────────────────

type salesChannelProductOverrideModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	ChannelID string `gorm:"not null;type:uuid"`
	ProductID string `gorm:"not null;type:uuid"`
	Included  bool   `gorm:"not null"`
	CreatedAt time.Time
}

func (salesChannelProductOverrideModel) TableName() string { return "sales_channel_product_overrides" }

type SalesChannelProductOverrideRepo struct{ db *gorm.DB }

func NewSalesChannelProductOverrideRepo(db *gorm.DB) *SalesChannelProductOverrideRepo {
	return &SalesChannelProductOverrideRepo{db: db}
}

func (r *SalesChannelProductOverrideRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert matches the unique constraint on (channel_id, product_id) -- a
// repeat add/remove action overwrites the prior override.
func (r *SalesChannelProductOverrideRepo) Upsert(ctx context.Context, o *channelsDomain.SalesChannelProductOverride) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}, {Name: "product_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"included"}),
	}).Create(&salesChannelProductOverrideModel{
		TenantID: o.TenantID, ChannelID: o.ChannelID, ProductID: o.ProductID, Included: o.Included, CreatedAt: o.CreatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("salesChannelProductOverrideRepo.Upsert: %w", err)
	}
	return nil
}

func (r *SalesChannelProductOverrideRepo) FindByChannelAndProduct(ctx context.Context, channelID, productID string) (*channelsDomain.SalesChannelProductOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesChannelProductOverrideModel
	if err := tx.Where("channel_id = ? AND product_id = ?", channelID, productID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("salesChannelProductOverrideRepo.FindByChannelAndProduct: %w", err)
	}
	return toSalesChannelProductOverrideDomain(m), nil
}

func (r *SalesChannelProductOverrideRepo) ListByChannel(ctx context.Context, channelID string) ([]channelsDomain.SalesChannelProductOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesChannelProductOverrideModel
	if err := tx.Where("channel_id = ?", channelID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesChannelProductOverrideRepo.ListByChannel: %w", err)
	}
	out := make([]channelsDomain.SalesChannelProductOverride, len(rows))
	for i, row := range rows {
		out[i] = *toSalesChannelProductOverrideDomain(row)
	}
	return out, nil
}

func (r *SalesChannelProductOverrideRepo) Delete(ctx context.Context, channelID, productID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("channel_id = ? AND product_id = ?", channelID, productID).Delete(&salesChannelProductOverrideModel{}).Error; err != nil {
		return fmt.Errorf("salesChannelProductOverrideRepo.Delete: %w", err)
	}
	return nil
}

func toSalesChannelProductOverrideDomain(m salesChannelProductOverrideModel) *channelsDomain.SalesChannelProductOverride {
	return &channelsDomain.SalesChannelProductOverride{
		ID: m.ID, TenantID: m.TenantID, ChannelID: m.ChannelID, ProductID: m.ProductID, Included: m.Included, CreatedAt: m.CreatedAt,
	}
}

func storefrontLocaleOrEnglish(l channelsDomain.StorefrontLocale) string {
	if l == "" {
		return string(channelsDomain.StorefrontLocaleEnglish)
	}
	return string(l)
}
