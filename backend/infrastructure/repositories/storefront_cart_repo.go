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

// storefrontCartItems is JSONB-backed, same driver.Valuer/sql.Scanner + whole-row-
// replace convention as whatsapp_conversation_repo.go's own storefrontCartItems type
// for whatsapp_conversations.cart.
type storefrontCartItems []storefrontDomain.CartItem

func (c storefrontCartItems) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]storefrontDomain.CartItem(c))
	return string(b), err
}

func (c *storefrontCartItems) Scan(value any) error {
	if value == nil {
		*c = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("storefrontCartItems.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]storefrontDomain.CartItem)(c))
}

type storefrontCartModel struct {
	ID             string              `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string              `gorm:"not null;type:uuid"`
	SalesChannelID string              `gorm:"column:sales_channel_id;not null;type:uuid"`
	Token          string              `gorm:"not null"`
	Items          storefrontCartItems `gorm:"type:jsonb;not null;default:'[]'"`
	CheckedOutAt   *time.Time
	CouponCode     *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (storefrontCartModel) TableName() string { return "storefront_carts" }

type StorefrontCartRepo struct{ db *gorm.DB }

func NewStorefrontCartRepo(db *gorm.DB) *StorefrontCartRepo { return &StorefrontCartRepo{db: db} }

func (r *StorefrontCartRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("storefrontCartRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *StorefrontCartRepo) Create(ctx context.Context, c *storefrontDomain.Cart) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := storefrontCartModel{
		TenantID: c.TenantID, SalesChannelID: c.SalesChannelID, Token: c.Token,
		Items: storefrontCartItems(c.Items), CheckedOutAt: c.CheckedOutAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("storefrontCartRepo.Create: %w", err)
	}
	c.ID, c.CreatedAt, c.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *StorefrontCartRepo) FindByToken(ctx context.Context, token string) (*storefrontDomain.Cart, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m storefrontCartModel
	err = tx.Where("token = ?", token).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("storefrontCartRepo.FindByToken: %w", err)
	}
	return toCartDomain(m), nil
}

func (r *StorefrontCartRepo) Update(ctx context.Context, c *storefrontDomain.Cart) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	result := tx.Model(&storefrontCartModel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"items":          storefrontCartItems(c.Items),
		"checked_out_at": c.CheckedOutAt,
		"coupon_code":    c.CouponCode,
		"updated_at":     c.UpdatedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("storefrontCartRepo.Update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return storefrontDomain.ErrCartNotFound
	}
	return nil
}

func toCartDomain(m storefrontCartModel) *storefrontDomain.Cart {
	return &storefrontDomain.Cart{
		ID: m.ID, TenantID: m.TenantID, SalesChannelID: m.SalesChannelID, Token: m.Token,
		Items: []storefrontDomain.CartItem(m.Items), CheckedOutAt: m.CheckedOutAt, CouponCode: m.CouponCode,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
