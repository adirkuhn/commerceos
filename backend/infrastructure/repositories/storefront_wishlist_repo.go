package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// storefrontWishlistProductIDs is JSONB-backed, same driver.Valuer/sql.Scanner
// + whole-row-replace convention as storefrontCartItems (storefront_cart_repo.go).
type storefrontWishlistProductIDs []string

func (p storefrontWishlistProductIDs) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]string(p))
	return string(b), err
}

func (p *storefrontWishlistProductIDs) Scan(value any) error {
	if value == nil {
		*p = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("storefrontWishlistProductIDs.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]string)(p))
}

type storefrontWishlistModel struct {
	ID             string                       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string                       `gorm:"not null;type:uuid"`
	SalesChannelID string                       `gorm:"column:sales_channel_id;not null;type:uuid"`
	Token          string                       `gorm:"not null"`
	ProductIDs     storefrontWishlistProductIDs `gorm:"column:product_ids;type:jsonb;not null;default:'[]'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (storefrontWishlistModel) TableName() string { return "storefront_wishlists" }

type StorefrontWishlistRepo struct{ db *gorm.DB }

func NewStorefrontWishlistRepo(db *gorm.DB) *StorefrontWishlistRepo {
	return &StorefrontWishlistRepo{db: db}
}

func (r *StorefrontWishlistRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("storefrontWishlistRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *StorefrontWishlistRepo) Create(ctx context.Context, w *storefrontDomain.Wishlist) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := storefrontWishlistModel{
		TenantID: w.TenantID, SalesChannelID: w.SalesChannelID, Token: w.Token,
		ProductIDs: storefrontWishlistProductIDs(w.ProductIDs),
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("storefrontWishlistRepo.Create: %w", err)
	}
	w.ID, w.CreatedAt, w.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *StorefrontWishlistRepo) FindByToken(ctx context.Context, token string) (*storefrontDomain.Wishlist, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m storefrontWishlistModel
	err = tx.Where("token = ?", token).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("storefrontWishlistRepo.FindByToken: %w", err)
	}
	return toWishlistDomain(m), nil
}

func (r *StorefrontWishlistRepo) Update(ctx context.Context, w *storefrontDomain.Wishlist) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	result := tx.Model(&storefrontWishlistModel{}).Where("id = ?", w.ID).Updates(map[string]any{
		"product_ids": storefrontWishlistProductIDs(w.ProductIDs),
		"updated_at":  w.UpdatedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("storefrontWishlistRepo.Update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return storefrontDomain.ErrWishlistNotFound
	}
	return nil
}

func toWishlistDomain(m storefrontWishlistModel) *storefrontDomain.Wishlist {
	return &storefrontDomain.Wishlist{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, Token: m.Token,
		ProductIDs: []string(m.ProductIDs), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
