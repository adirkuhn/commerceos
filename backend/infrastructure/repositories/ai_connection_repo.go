package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type aiConnectionModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	Provider        string `gorm:"not null"`
	APIKeyEncrypted string `gorm:"column:api_key_encrypted"`
	BaseURL         string `gorm:"column:base_url"`
	Model           string `gorm:"not null;default:''"`
	FallbackModels  string `gorm:"column:fallback_models;not null;default:''"`
	Status          string `gorm:"not null;default:'connected'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (aiConnectionModel) TableName() string { return "ai_connections" }

type AIConnectionRepo struct{ db *gorm.DB }

func NewAIConnectionRepo(db *gorm.DB) *AIConnectionRepo { return &AIConnectionRepo{db: db} }

func (r *AIConnectionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("integrations: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *AIConnectionRepo) Get(ctx context.Context) (*integrationsDomain.AIConnection, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m aiConnectionModel
	if err := tx.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("aiConnectionRepo.Get: %w", err)
	}
	return toAIConnectionDomain(m), nil
}

func (r *AIConnectionRepo) Upsert(ctx context.Context, c *integrationsDomain.AIConnection) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := aiConnectionModel{
		TenantID: c.TenantID, Provider: string(c.Provider),
		APIKeyEncrypted: c.APIKeyEncrypted, BaseURL: c.BaseURL, Model: c.Model, FallbackModels: c.FallbackModels,
		Status: string(c.Status), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider", "api_key_encrypted", "base_url", "model", "fallback_models", "status", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("aiConnectionRepo.Upsert: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *AIConnectionRepo) Delete(ctx context.Context) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	// "1 = 1" only satisfies GORM's guard against a condition-less global
	// delete -- RLS (tenant.WithTenant's SET LOCAL app.current_tenant) is
	// what actually scopes this to the current tenant's single row.
	if err := tx.Where("1 = 1").Delete(&aiConnectionModel{}).Error; err != nil {
		return fmt.Errorf("aiConnectionRepo.Delete: %w", err)
	}
	return nil
}

func toAIConnectionDomain(m aiConnectionModel) *integrationsDomain.AIConnection {
	return &integrationsDomain.AIConnection{
		ID: m.ID, TenantID: m.TenantID, Provider: integrationsDomain.AIProvider(m.Provider),
		APIKeyEncrypted: m.APIKeyEncrypted, BaseURL: m.BaseURL, Model: m.Model, FallbackModels: m.FallbackModels,
		Status: integrationsDomain.Status(m.Status), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
