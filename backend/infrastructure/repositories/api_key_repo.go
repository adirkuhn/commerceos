package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type apiKeyModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string `gorm:"not null;type:uuid"`
	Name        string `gorm:"not null"`
	Description string `gorm:"not null;default:''"`
	KeyHash     string `gorm:"column:key_hash;not null"`
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

func (apiKeyModel) TableName() string { return "storefront_api_keys" }

type APIKeyRepo struct{ db *gorm.DB }

func NewAPIKeyRepo(db *gorm.DB) *APIKeyRepo { return &APIKeyRepo{db: db} }

func (r *APIKeyRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("apiKeyRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *APIKeyRepo) Create(ctx context.Context, k *apikeysDomain.APIKey) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := apiKeyModel{
		TenantID: k.TenantID, Name: k.Name, Description: k.Description, KeyHash: k.KeyHash,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("apiKeyRepo.Create: %w", err)
	}
	k.ID, k.CreatedAt = m.ID, m.CreatedAt
	return nil
}

func (r *APIKeyRepo) ListByTenant(ctx context.Context) ([]apikeysDomain.APIKey, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []apiKeyModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("apiKeyRepo.ListByTenant: %w", err)
	}
	out := make([]apikeysDomain.APIKey, len(rows))
	for i, row := range rows {
		out[i] = toAPIKeyDomain(row)
	}
	return out, nil
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	result := tx.Model(&apiKeyModel{}).Where("id = ?", id).Update("revoked_at", time.Now())
	if result.Error != nil {
		return fmt.Errorf("apiKeyRepo.Revoke: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apikeysDomain.ErrNotFound
	}
	return nil
}

func (r *APIKeyRepo) FindActiveTenantIDByHash(ctx context.Context, hash string) (string, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return "", err
	}
	var m apiKeyModel
	if err := tx.Where("key_hash = ? AND revoked_at IS NULL", hash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apikeysDomain.ErrNotFound
		}
		return "", fmt.Errorf("apiKeyRepo.FindActiveTenantIDByHash: %w", err)
	}
	return m.TenantID, nil
}

func toAPIKeyDomain(m apiKeyModel) apikeysDomain.APIKey {
	return apikeysDomain.APIKey{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, Description: m.Description,
		KeyHash: m.KeyHash, CreatedAt: m.CreatedAt, RevokedAt: m.RevokedAt,
	}
}
