package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type whatsappMessageModel struct {
	ID         string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string  `gorm:"not null;type:uuid"`
	PhoneE164  string  `gorm:"column:phone_e164;not null"`
	CustomerID *string `gorm:"column:customer_id;type:uuid"`
	Direction  string  `gorm:"not null"`
	Body       string  `gorm:"not null"`
	CreatedAt  time.Time
}

func (whatsappMessageModel) TableName() string { return "whatsapp_messages" }

type WhatsAppMessageRepo struct{ db *gorm.DB }

func NewWhatsAppMessageRepo(db *gorm.DB) *WhatsAppMessageRepo { return &WhatsAppMessageRepo{db: db} }

func (r *WhatsAppMessageRepo) Create(ctx context.Context, m *whatsappDomain.Message) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("whatsappMessageRepo: operation called outside a transaction")
	}
	row := whatsappMessageModel{
		TenantID: m.TenantID, PhoneE164: m.PhoneE164, CustomerID: m.CustomerID,
		Direction: string(m.Direction), Body: m.Body,
	}
	if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("whatsappMessageRepo.Create: %w", err)
	}
	m.ID = row.ID
	m.CreatedAt = row.CreatedAt
	return nil
}

func (r *WhatsAppMessageRepo) ListByPhone(ctx context.Context, phoneE164 string) ([]whatsappDomain.Message, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("whatsappMessageRepo: operation called outside a transaction")
	}
	var rows []whatsappMessageModel
	if err := tx.WithContext(ctx).Where("phone_e164 = ?", phoneE164).
		Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("whatsappMessageRepo.ListByPhone: %w", err)
	}
	return toDomainWhatsAppMessages(rows), nil
}

func (r *WhatsAppMessageRepo) ListByCustomer(ctx context.Context, customerID string) ([]whatsappDomain.Message, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("whatsappMessageRepo: operation called outside a transaction")
	}
	var rows []whatsappMessageModel
	if err := tx.WithContext(ctx).Where("customer_id = ?", customerID).
		Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("whatsappMessageRepo.ListByCustomer: %w", err)
	}
	return toDomainWhatsAppMessages(rows), nil
}

func toDomainWhatsAppMessages(rows []whatsappMessageModel) []whatsappDomain.Message {
	out := make([]whatsappDomain.Message, len(rows))
	for i, r := range rows {
		out[i] = whatsappDomain.Message{
			ID: r.ID, TenantID: r.TenantID, PhoneE164: r.PhoneE164, CustomerID: r.CustomerID,
			Direction: whatsappDomain.MessageDirection(r.Direction), Body: r.Body, CreatedAt: r.CreatedAt,
		}
	}
	return out
}
