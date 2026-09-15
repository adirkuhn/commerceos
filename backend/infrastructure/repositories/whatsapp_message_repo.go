package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type whatsappMessageModel struct {
	ID             string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string  `gorm:"not null;type:uuid"`
	PhoneE164      string  `gorm:"column:phone_e164;not null"`
	CustomerID     *string `gorm:"column:customer_id;type:uuid"`
	Direction      string  `gorm:"not null"`
	Body           string  `gorm:"not null"`
	WebhookEventID *string `gorm:"column:webhook_event_id"`
	CreatedAt      time.Time
}

func (whatsappMessageModel) TableName() string { return "whatsapp_messages" }

type WhatsAppMessageRepo struct{ db *gorm.DB }

func NewWhatsAppMessageRepo(db *gorm.DB) *WhatsAppMessageRepo { return &WhatsAppMessageRepo{db: db} }

// Create dedupes on (tenant_id, webhook_event_id) via ON CONFLICT DO
// NOTHING -- safe to apply unconditionally (not just when m.WebhookEventID
// is set) because Postgres never treats two NULLs as conflicting in a
// unique index, so an outbound reply or any other message with no
// WebhookEventID always inserts normally. A skipped insert (WebhookEventID
// already logged, e.g. this NATS delivery is a retry of an envelope
// already recorded) leaves m.ID/m.CreatedAt at their zero value -- callers
// (HandleInboundMessageUseCase) don't use them afterward for the inbound
// row, only for logging.
func (r *WhatsAppMessageRepo) Create(ctx context.Context, m *whatsappDomain.Message) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("whatsappMessageRepo: operation called outside a transaction")
	}
	row := whatsappMessageModel{
		TenantID: m.TenantID, PhoneE164: m.PhoneE164, CustomerID: m.CustomerID,
		Direction: string(m.Direction), Body: m.Body, WebhookEventID: m.WebhookEventID,
	}
	err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "webhook_event_id"}}, DoNothing: true,
	}).Create(&row).Error
	if err != nil {
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

func (r *WhatsAppMessageRepo) DeleteByPhone(ctx context.Context, phoneE164 string) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("whatsappMessageRepo: operation called outside a transaction")
	}
	if err := tx.WithContext(ctx).Where("phone_e164 = ?", phoneE164).Delete(&whatsappMessageModel{}).Error; err != nil {
		return fmt.Errorf("whatsappMessageRepo.DeleteByPhone: %w", err)
	}
	return nil
}

// conversationSummaryRow is ListConversations' raw query result shape --
// FirstName/LastName come from a LEFT JOIN against customers (NULL when
// CustomerID is nil or the row somehow no longer matches one), combined
// into ConversationSummary.CustomerName in toDomainConversationSummaries.
type conversationSummaryRow struct {
	PhoneE164     string
	CustomerID    *string
	FirstName     *string
	LastName      *string
	LastDirection string
	LastBody      string
	LastMessageAt time.Time
}

// ListConversations picks each phone number's single most recent message
// via Postgres's DISTINCT ON (phone_e164 ... ORDER BY phone_e164,
// created_at DESC), then re-sorts that one-row-per-phone result by
// created_at DESC in an outer query -- DISTINCT ON requires phone_e164 to
// lead the ORDER BY for the dedup itself to pick the right row, so the
// conversation list's actual newest-first order has to be a second pass
// over the deduped set, not the same ORDER BY.
func (r *WhatsAppMessageRepo) ListConversations(ctx context.Context) ([]whatsappDomain.ConversationSummary, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("whatsappMessageRepo: operation called outside a transaction")
	}
	var rows []conversationSummaryRow
	err := tx.WithContext(ctx).Raw(`
		SELECT * FROM (
			SELECT DISTINCT ON (wm.phone_e164)
				wm.phone_e164 AS phone_e164, wm.customer_id AS customer_id,
				c.first_name AS first_name, c.last_name AS last_name,
				wm.direction AS last_direction, wm.body AS last_body, wm.created_at AS last_message_at
			FROM whatsapp_messages wm
			LEFT JOIN customers c ON c.id = wm.customer_id
			ORDER BY wm.phone_e164, wm.created_at DESC
		) conversations
		ORDER BY last_message_at DESC
	`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("whatsappMessageRepo.ListConversations: %w", err)
	}
	return toDomainConversationSummaries(rows), nil
}

func toDomainConversationSummaries(rows []conversationSummaryRow) []whatsappDomain.ConversationSummary {
	out := make([]whatsappDomain.ConversationSummary, len(rows))
	for i, r := range rows {
		name := ""
		if r.FirstName != nil {
			name = *r.FirstName
			if r.LastName != nil && *r.LastName != "" {
				name += " " + *r.LastName
			}
		}
		out[i] = whatsappDomain.ConversationSummary{
			PhoneE164: r.PhoneE164, CustomerID: r.CustomerID, CustomerName: name,
			LastDirection: whatsappDomain.MessageDirection(r.LastDirection), LastBody: r.LastBody, LastMessageAt: r.LastMessageAt,
		}
	}
	return out
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
