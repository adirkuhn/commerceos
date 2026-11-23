package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type aiConnectionModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	Position        int    `gorm:"column:position;not null"`
	Label           string `gorm:"column:label;not null"`
	Enabled         bool   `gorm:"column:enabled;not null"`
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

// AIConnectionRepo implements integrations.AIConnectionRepository.
type AIConnectionRepo struct{ db *gorm.DB }

func NewAIConnectionRepo(db *gorm.DB) *AIConnectionRepo { return &AIConnectionRepo{db: db} }

func (r *AIConnectionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("integrations: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *AIConnectionRepo) List(ctx context.Context) ([]integrationsDomain.AIConnection, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []aiConnectionModel
	if err := tx.Order("position, created_at, id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("aiConnectionRepo.List: %w", err)
	}
	out := make([]integrationsDomain.AIConnection, len(rows))
	for i, m := range rows {
		out[i] = *toAIConnectionDomain(m)
	}
	return out, nil
}

func (r *AIConnectionRepo) Get(ctx context.Context, id string) (*integrationsDomain.AIConnection, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m aiConnectionModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, integrationsDomain.ErrAIConnectionNotFound
		}
		return nil, fmt.Errorf("aiConnectionRepo.Get: %w", err)
	}
	return toAIConnectionDomain(m), nil
}

func (r *AIConnectionRepo) Create(ctx context.Context, c *integrationsDomain.AIConnection) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toAIConnectionModel(*c)
	m.ID = ""
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("aiConnectionRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *AIConnectionRepo) Update(ctx context.Context, c *integrationsDomain.AIConnection) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toAIConnectionModel(*c)
	res := tx.Model(&aiConnectionModel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"position": m.Position, "label": m.Label, "enabled": m.Enabled, "provider": m.Provider,
		"api_key_encrypted": m.APIKeyEncrypted, "base_url": m.BaseURL, "model": m.Model,
		"fallback_models": m.FallbackModels, "status": m.Status, "updated_at": m.UpdatedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("aiConnectionRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return integrationsDomain.ErrAIConnectionNotFound
	}
	return nil
}

func (r *AIConnectionRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&aiConnectionModel{})
	if res.Error != nil {
		return fmt.Errorf("aiConnectionRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return integrationsDomain.ErrAIConnectionNotFound
	}
	return nil
}

func toAIConnectionModel(c integrationsDomain.AIConnection) aiConnectionModel {
	return aiConnectionModel{
		ID: c.ID, TenantID: c.TenantID, Position: c.Position, Label: c.Label, Enabled: c.Enabled,
		Provider: string(c.Provider), APIKeyEncrypted: c.APIKeyEncrypted, BaseURL: c.BaseURL, Model: c.Model,
		FallbackModels: c.FallbackModels, Status: string(c.Status), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toAIConnectionDomain(m aiConnectionModel) *integrationsDomain.AIConnection {
	return &integrationsDomain.AIConnection{
		ID: m.ID, TenantID: m.TenantID, Position: m.Position, Label: m.Label, Enabled: m.Enabled,
		Provider: integrationsDomain.AIProvider(m.Provider), APIKeyEncrypted: m.APIKeyEncrypted, BaseURL: m.BaseURL,
		Model: m.Model, FallbackModels: m.FallbackModels, Status: integrationsDomain.Status(m.Status),
		CreatedAt: m.CreatedAt.UTC(), UpdatedAt: m.UpdatedAt.UTC(),
	}
}
