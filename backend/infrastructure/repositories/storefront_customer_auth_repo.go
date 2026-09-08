package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type storefrontMagicLinkTokenModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string `gorm:"column:tenant_id;not null;type:uuid"`
	SalesChannelID string `gorm:"column:sales_channel_id;not null;type:uuid"`
	CustomerID     string `gorm:"column:customer_id;not null;type:uuid"`
	Token          string `gorm:"not null"`
	ExpiresAt      time.Time
	UsedAt         *time.Time
	CreatedAt      time.Time
}

func (storefrontMagicLinkTokenModel) TableName() string { return "storefront_magic_link_tokens" }

type StorefrontMagicLinkTokenRepo struct{ db *gorm.DB }

func NewStorefrontMagicLinkTokenRepo(db *gorm.DB) *StorefrontMagicLinkTokenRepo {
	return &StorefrontMagicLinkTokenRepo{db: db}
}

func (r *StorefrontMagicLinkTokenRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("storefrontMagicLinkTokenRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *StorefrontMagicLinkTokenRepo) Create(ctx context.Context, t *storefrontDomain.MagicLinkToken) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := storefrontMagicLinkTokenModel{
		TenantID: t.TenantID, SalesChannelID: t.SalesChannelID, CustomerID: t.CustomerID,
		Token: t.Token, ExpiresAt: t.ExpiresAt, UsedAt: t.UsedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("storefrontMagicLinkTokenRepo.Create: %w", err)
	}
	t.ID, t.CreatedAt = m.ID, m.CreatedAt
	return nil
}

func (r *StorefrontMagicLinkTokenRepo) FindByToken(ctx context.Context, token string) (*storefrontDomain.MagicLinkToken, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m storefrontMagicLinkTokenModel
	err = tx.Where("token = ?", token).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("storefrontMagicLinkTokenRepo.FindByToken: %w", err)
	}
	return &storefrontDomain.MagicLinkToken{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, CustomerID: m.CustomerID,
		Token: m.Token, ExpiresAt: m.ExpiresAt, UsedAt: m.UsedAt, CreatedAt: m.CreatedAt,
	}, nil
}

// MarkUsed only succeeds against a still-unused row -- see
// storefrontDomain.MagicLinkTokenRepository.MarkUsed's doc comment on why
// this must be atomic, not read-modify-write.
func (r *StorefrontMagicLinkTokenRepo) MarkUsed(ctx context.Context, id string, usedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	result := tx.Model(&storefrontMagicLinkTokenModel{}).Where("id = ? AND used_at IS NULL", id).Update("used_at", usedAt)
	if result.Error != nil {
		return fmt.Errorf("storefrontMagicLinkTokenRepo.MarkUsed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return storefrontDomain.ErrInvalidOrExpiredMagicLink
	}
	return nil
}

type storefrontCustomerSessionModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string `gorm:"column:tenant_id;not null;type:uuid"`
	SalesChannelID string `gorm:"column:sales_channel_id;not null;type:uuid"`
	CustomerID     string `gorm:"column:customer_id;not null;type:uuid"`
	Token          string `gorm:"not null"`
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

func (storefrontCustomerSessionModel) TableName() string { return "storefront_customer_sessions" }

type StorefrontCustomerSessionRepo struct{ db *gorm.DB }

func NewStorefrontCustomerSessionRepo(db *gorm.DB) *StorefrontCustomerSessionRepo {
	return &StorefrontCustomerSessionRepo{db: db}
}

func (r *StorefrontCustomerSessionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("storefrontCustomerSessionRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *StorefrontCustomerSessionRepo) Create(ctx context.Context, s *storefrontDomain.CustomerSession) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := storefrontCustomerSessionModel{
		TenantID: s.TenantID, SalesChannelID: s.SalesChannelID, CustomerID: s.CustomerID,
		Token: s.Token, ExpiresAt: s.ExpiresAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("storefrontCustomerSessionRepo.Create: %w", err)
	}
	s.ID, s.CreatedAt = m.ID, m.CreatedAt
	return nil
}

func (r *StorefrontCustomerSessionRepo) FindByToken(ctx context.Context, token string) (*storefrontDomain.CustomerSession, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m storefrontCustomerSessionModel
	err = tx.Where("token = ?", token).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("storefrontCustomerSessionRepo.FindByToken: %w", err)
	}
	return &storefrontDomain.CustomerSession{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, CustomerID: m.CustomerID,
		Token: m.Token, ExpiresAt: m.ExpiresAt, CreatedAt: m.CreatedAt,
	}, nil
}

func (r *StorefrontCustomerSessionRepo) Delete(ctx context.Context, token string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("token = ?", token).Delete(&storefrontCustomerSessionModel{}).Error; err != nil {
		return fmt.Errorf("storefrontCustomerSessionRepo.Delete: %w", err)
	}
	return nil
}
