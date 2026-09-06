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

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// cartItems is JSONB-backed, same driver.Valuer/sql.Scanner convention as
// pms_repo.go's stringSlice.
type cartItems []whatsappDomain.CartItem

func (c cartItems) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]whatsappDomain.CartItem(c))
	return string(b), err
}

func (c *cartItems) Scan(value any) error {
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
		return fmt.Errorf("cartItems.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]whatsappDomain.CartItem)(c))
}

type whatsappConversationModel struct {
	ID                     string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID               string    `gorm:"not null;type:uuid"`
	PhoneE164              string    `gorm:"not null"`
	State                  string    `gorm:"not null"`
	Cart                   cartItems `gorm:"type:jsonb;not null;default:'[]'"`
	PendingCustomerName    string    `gorm:"not null"`
	PendingCustomerEmail   string    `gorm:"not null"`
	PendingCustomerAddress string    `gorm:"not null"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (whatsappConversationModel) TableName() string { return "whatsapp_conversations" }

type WhatsAppConversationRepo struct{ db *gorm.DB }

func NewWhatsAppConversationRepo(db *gorm.DB) *WhatsAppConversationRepo {
	return &WhatsAppConversationRepo{db: db}
}

func (r *WhatsAppConversationRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("whatsappConversationRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *WhatsAppConversationRepo) FindByPhone(ctx context.Context, phoneE164 string) (*whatsappDomain.Conversation, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m whatsappConversationModel
	err = tx.Where("phone_e164 = ?", phoneE164).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("whatsappConversationRepo.FindByPhone: %w", err)
	}
	return toConversationDomain(m), nil
}

// Upsert inserts a new row (leaving ID for Postgres's own gen_random_uuid()
// default -- never set explicitly, same as every other Create in this
// codebase), or updates every mutable column when (tenant_id, phone_e164)
// already exists -- one conversation per phone number, same convention as
// channels.SalesChannelProductOverrideRepository.Upsert.
func (r *WhatsAppConversationRepo) Upsert(ctx context.Context, c *whatsappDomain.Conversation) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "phone_e164"}},
		DoUpdates: clause.AssignmentColumns([]string{"state", "cart", "pending_customer_name", "pending_customer_email", "pending_customer_address", "updated_at"}),
	}).Create(&whatsappConversationModel{
		TenantID: c.TenantID, PhoneE164: c.PhoneE164, State: string(c.State),
		Cart: cartItems(c.Cart), PendingCustomerName: c.PendingCustomerName,
		PendingCustomerEmail: c.PendingCustomerEmail, PendingCustomerAddress: c.PendingCustomerAddress,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("whatsappConversationRepo.Upsert: %w", err)
	}
	return nil
}

func toConversationDomain(m whatsappConversationModel) *whatsappDomain.Conversation {
	return &whatsappDomain.Conversation{
		ID: m.ID, TenantID: m.TenantID, PhoneE164: m.PhoneE164, State: whatsappDomain.ConversationState(m.State),
		Cart: []whatsappDomain.CartItem(m.Cart), PendingCustomerName: m.PendingCustomerName,
		PendingCustomerEmail: m.PendingCustomerEmail, PendingCustomerAddress: m.PendingCustomerAddress,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
