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

type integrationConnectionModel struct {
	ID                      string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                string  `gorm:"not null;type:uuid"`
	Provider                string  `gorm:"not null"`
	StoreURL                string  `gorm:"not null"`
	ConsumerKey             string  `gorm:"not null"`
	ConsumerSecretEncrypted string  `gorm:"not null"`
	WebhookSecretEncrypted  string  `gorm:"not null"`
	Status                  string  `gorm:"not null"`
	DefaultCurrency         string  `gorm:"not null"`
	DefaultWarehouseID      *string `gorm:"type:uuid"`
	BolEconomicOperatorID   *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (integrationConnectionModel) TableName() string { return "integration_connections" }

type ConnectionRepo struct{ db *gorm.DB }

func NewConnectionRepo(db *gorm.DB) *ConnectionRepo { return &ConnectionRepo{db: db} }

func (r *ConnectionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("integrations: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ConnectionRepo) Create(ctx context.Context, c *integrationsDomain.Connection) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := integrationConnectionModel{
		TenantID:                c.TenantID,
		Provider:                string(c.Provider),
		StoreURL:                c.StoreURL.String(),
		ConsumerKey:             c.ConsumerKey,
		ConsumerSecretEncrypted: c.ConsumerSecretEncrypted,
		WebhookSecretEncrypted:  c.WebhookSecretEncrypted,
		Status:                  string(c.Status),
		DefaultCurrency:         c.DefaultCurrency,
		DefaultWarehouseID:      c.DefaultWarehouseID,
		BolEconomicOperatorID:   c.BolEconomicOperatorID,
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("connectionRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *ConnectionRepo) FindByID(ctx context.Context, id string) (*integrationsDomain.Connection, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m integrationConnectionModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, integrationsDomain.ErrConnectionNotFound
		}
		return nil, fmt.Errorf("connectionRepo.FindByID: %w", err)
	}
	return toConnectionDomain(m), nil
}

func (r *ConnectionRepo) ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []integrationConnectionModel
	if err := tx.Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("connectionRepo.ListByTenant: %w", err)
	}
	conns := make([]integrationsDomain.Connection, len(rows))
	for i, row := range rows {
		conns[i] = *toConnectionDomain(row)
	}
	return conns, nil
}

func (r *ConnectionRepo) Update(ctx context.Context, c *integrationsDomain.Connection) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&integrationConnectionModel{}).Where("id = ?", c.ID).
		Updates(map[string]any{
			"status":                   string(c.Status),
			"default_currency":         c.DefaultCurrency,
			"default_warehouse_id":     c.DefaultWarehouseID,
			"webhook_secret_encrypted": c.WebhookSecretEncrypted,
			"bol_economic_operator_id": c.BolEconomicOperatorID,
			"updated_at":               c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("connectionRepo.Update: %w", err)
	}
	return nil
}

func (r *ConnectionRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&integrationConnectionModel{}).Error; err != nil {
		return fmt.Errorf("connectionRepo.Delete: %w", err)
	}
	return nil
}

func toConnectionDomain(m integrationConnectionModel) *integrationsDomain.Connection {
	return &integrationsDomain.Connection{
		ID:                      m.ID,
		TenantID:                m.TenantID,
		Provider:                integrationsDomain.Provider(m.Provider),
		StoreURL:                integrationsDomain.StoreURL(m.StoreURL),
		ConsumerKey:             m.ConsumerKey,
		ConsumerSecretEncrypted: m.ConsumerSecretEncrypted,
		WebhookSecretEncrypted:  m.WebhookSecretEncrypted,
		Status:                  integrationsDomain.Status(m.Status),
		DefaultCurrency:         m.DefaultCurrency,
		DefaultWarehouseID:      m.DefaultWarehouseID,
		BolEconomicOperatorID:   m.BolEconomicOperatorID,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}
}
