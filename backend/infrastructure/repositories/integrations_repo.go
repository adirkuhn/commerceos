package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type integrationConnectionModel struct {
	ID                        string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                  string  `gorm:"not null;type:uuid"`
	Provider                  string  `gorm:"not null"`
	StoreURL                  string  `gorm:"not null"`
	ConsumerKey               string  `gorm:"not null"`
	ConsumerSecretEncrypted   string  `gorm:"not null"`
	WebhookSecretEncrypted    string  `gorm:"not null"`
	Status                    string  `gorm:"not null"`
	DefaultCurrency           string  `gorm:"not null"`
	DefaultWarehouseID        *string `gorm:"type:uuid"`
	BolEconomicOperatorID     *string
	BolDeliveryCode           *string
	BolShippingCostMinorUnits int64 `gorm:"not null"`
	SMTPPort                  *string
	EmailFrom                 *string
	// Explicit column names: GORM's default snake_case conversion splits
	// "WhatsApp" into "whats_app" (not a recognized single word/acronym to
	// it), which doesn't match what migration 172 actually created --
	// verified against a real production error before this fix
	// ("column \"whats_app_test_mode_enabled\" ... does not exist").
	WhatsAppTestModeEnabled  bool    `gorm:"column:whatsapp_test_mode_enabled;not null;default:false"`
	WhatsAppTestPhoneE164    *string `gorm:"column:whatsapp_test_phone_e164"`
	WhatsAppAIParsingEnabled bool    `gorm:"column:whatsapp_ai_parsing_enabled;not null;default:false"`
	// Explicit column names for the same reason as WhatsApp above: the
	// default namer would split "PostNL" into "post_nl".
	PostNLCustomerCode   string `gorm:"column:postnl_customer_code;not null;default:''"`
	PostNLCustomerNumber string `gorm:"column:postnl_customer_number;not null;default:''"`
	PostNLSandbox        bool   `gorm:"column:postnl_sandbox;not null;default:false"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
		TenantID:                  c.TenantID,
		Provider:                  string(c.Provider),
		StoreURL:                  c.StoreURL.String(),
		ConsumerKey:               c.ConsumerKey,
		ConsumerSecretEncrypted:   c.ConsumerSecretEncrypted,
		WebhookSecretEncrypted:    c.WebhookSecretEncrypted,
		Status:                    string(c.Status),
		DefaultCurrency:           c.DefaultCurrency,
		DefaultWarehouseID:        c.DefaultWarehouseID,
		BolEconomicOperatorID:     c.BolEconomicOperatorID,
		BolDeliveryCode:           c.BolDeliveryCode,
		BolShippingCostMinorUnits: c.BolShippingCostMinorUnits,
		SMTPPort:                  c.SMTPPort,
		EmailFrom:                 c.EmailFrom,
		WhatsAppTestModeEnabled:   c.WhatsAppTestModeEnabled,
		WhatsAppTestPhoneE164:     c.WhatsAppTestPhoneE164,
		WhatsAppAIParsingEnabled:  c.WhatsAppAIParsingEnabled,
		PostNLCustomerCode:        c.PostNLCustomerCode,
		PostNLCustomerNumber:      c.PostNLCustomerNumber,
		PostNLSandbox:             c.PostNLSandbox,
		CreatedAt:                 c.CreatedAt,
		UpdatedAt:                 c.UpdatedAt,
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
	if err := tx.Where("id = ? AND status <> ?", id, integrationsDomain.StatusDisconnected).First(&m).Error; err != nil {
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
	if err := tx.Where("status <> ?", integrationsDomain.StatusDisconnected).Order("created_at asc").Find(&rows).Error; err != nil {
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
			"status":                        string(c.Status),
			"default_currency":              c.DefaultCurrency,
			"default_warehouse_id":          c.DefaultWarehouseID,
			"store_url":                     c.StoreURL.String(),
			"consumer_key":                  c.ConsumerKey,
			"consumer_secret_encrypted":     c.ConsumerSecretEncrypted,
			"webhook_secret_encrypted":      c.WebhookSecretEncrypted,
			"bol_economic_operator_id":      c.BolEconomicOperatorID,
			"bol_delivery_code":             c.BolDeliveryCode,
			"bol_shipping_cost_minor_units": c.BolShippingCostMinorUnits,
			"whatsapp_test_mode_enabled":    c.WhatsAppTestModeEnabled,
			"whatsapp_test_phone_e164":      c.WhatsAppTestPhoneE164,
			"whatsapp_ai_parsing_enabled":   c.WhatsAppAIParsingEnabled,
			"postnl_customer_code":          c.PostNLCustomerCode,
			"postnl_customer_number":        c.PostNLCustomerNumber,
			"postnl_sandbox":                c.PostNLSandbox,
			"smtp_port":                     c.SMTPPort,
			"email_from":                    c.EmailFrom,
			"updated_at":                    c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("connectionRepo.Update: %w", err)
	}
	return nil
}

func (r *ConnectionRepo) FindByStore(ctx context.Context, provider integrationsDomain.Provider, storeURL integrationsDomain.StoreURL) (*integrationsDomain.Connection, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m integrationConnectionModel
	err = tx.Where("provider = ? AND "+sameStoreSQL("store_url")+" = "+sameStoreSQL("?::text"),
		string(provider), strings.TrimSpace(storeURL.String())).
		Order(clause.OrderBy{Expression: clause.Expr{SQL: "(status = ?), updated_at DESC", Vars: []any{string(integrationsDomain.StatusDisconnected)}}}).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, integrationsDomain.ErrConnectionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("connectionRepo.FindByStore: %w", err)
	}
	return toConnectionDomain(m), nil
}

// sameStoreSQL reduces a store URL to what identifies the store: no scheme,
// no www., no case, no trailing slash. Another subdomain stays another store.
// The pattern has no question marks: GORM would read them as bind variables.
func sameStoreSQL(expr string) string {
	return "regexp_replace(lower(rtrim(" + expr + ", '/')), '^(http://|https://|)(www[.]|)', '')"
}

func toConnectionDomain(m integrationConnectionModel) *integrationsDomain.Connection {
	return &integrationsDomain.Connection{
		ID:                        m.ID,
		TenantID:                  m.TenantID,
		Provider:                  integrationsDomain.Provider(m.Provider),
		StoreURL:                  integrationsDomain.StoreURL(m.StoreURL),
		ConsumerKey:               m.ConsumerKey,
		ConsumerSecretEncrypted:   m.ConsumerSecretEncrypted,
		WebhookSecretEncrypted:    m.WebhookSecretEncrypted,
		Status:                    integrationsDomain.Status(m.Status),
		DefaultCurrency:           m.DefaultCurrency,
		DefaultWarehouseID:        m.DefaultWarehouseID,
		BolEconomicOperatorID:     m.BolEconomicOperatorID,
		BolDeliveryCode:           m.BolDeliveryCode,
		BolShippingCostMinorUnits: m.BolShippingCostMinorUnits,
		SMTPPort:                  m.SMTPPort,
		EmailFrom:                 m.EmailFrom,
		WhatsAppTestModeEnabled:   m.WhatsAppTestModeEnabled,
		WhatsAppTestPhoneE164:     m.WhatsAppTestPhoneE164,
		WhatsAppAIParsingEnabled:  m.WhatsAppAIParsingEnabled,
		PostNLCustomerCode:        m.PostNLCustomerCode,
		PostNLCustomerNumber:      m.PostNLCustomerNumber,
		PostNLSandbox:             m.PostNLSandbox,
		CreatedAt:                 m.CreatedAt,
		UpdatedAt:                 m.UpdatedAt,
	}
}
