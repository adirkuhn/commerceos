package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// messageTemplateChannels is a zero-dependency []messaging.Channel <-> jsonb
// Scanner/Valuer, same convention as pms_repo.go's stringSlice.
type messageTemplateChannels []messaging.Channel

func (c messageTemplateChannels) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]messaging.Channel(c))
	return string(b), err
}

func (c *messageTemplateChannels) Scan(value any) error {
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
		return fmt.Errorf("messageTemplateChannels.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]messaging.Channel)(c))
}

type messageTemplateModel struct {
	ID        string                  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string                  `gorm:"column:tenant_id;not null;type:uuid"`
	Name      string                  `gorm:"column:name;not null"`
	Category  string                  `gorm:"column:category;not null"`
	Channels  messageTemplateChannels `gorm:"column:channels;type:jsonb;not null"`
	Subject   string                  `gorm:"column:subject;not null"`
	Body      string                  `gorm:"column:body;not null"`
	Enabled   bool                    `gorm:"column:enabled;not null"`
	CreatedAt time.Time               `gorm:"column:created_at"`
	UpdatedAt time.Time               `gorm:"column:updated_at"`
}

func (messageTemplateModel) TableName() string { return "message_templates" }

type MessageTemplateRepo struct{ db *gorm.DB }

func NewMessageTemplateRepo(db *gorm.DB) *MessageTemplateRepo { return &MessageTemplateRepo{db: db} }

func (r *MessageTemplateRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("messageTemplateRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *MessageTemplateRepo) Create(ctx context.Context, t *mcDomain.MessageTemplate) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toMessageTemplateModel(*t)
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("messageTemplateRepo.Create: %w", err)
	}
	t.ID = m.ID
	return nil
}

func (r *MessageTemplateRepo) FindByID(ctx context.Context, id string) (*mcDomain.MessageTemplate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m messageTemplateModel
	err = tx.Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, mcDomain.ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messageTemplateRepo.FindByID: %w", err)
	}
	out := toMessageTemplateDomain(m)
	return &out, nil
}

func (r *MessageTemplateRepo) ListByTenant(ctx context.Context) ([]mcDomain.MessageTemplate, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []messageTemplateModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("messageTemplateRepo.ListByTenant: %w", err)
	}
	out := make([]mcDomain.MessageTemplate, len(rows))
	for i, row := range rows {
		out[i] = toMessageTemplateDomain(row)
	}
	return out, nil
}

func (r *MessageTemplateRepo) Update(ctx context.Context, t *mcDomain.MessageTemplate) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toMessageTemplateModel(*t)
	if err := tx.Model(&messageTemplateModel{}).Where("id = ?", t.ID).Updates(map[string]any{
		"name": m.Name, "category": m.Category, "channels": m.Channels,
		"subject": m.Subject, "body": m.Body, "enabled": m.Enabled, "updated_at": m.UpdatedAt,
	}).Error; err != nil {
		return fmt.Errorf("messageTemplateRepo.Update: %w", err)
	}
	return nil
}

func (r *MessageTemplateRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&messageTemplateModel{}).Error; err != nil {
		return fmt.Errorf("messageTemplateRepo.Delete: %w", err)
	}
	return nil
}

func toMessageTemplateModel(t mcDomain.MessageTemplate) messageTemplateModel {
	return messageTemplateModel{
		ID: t.ID, TenantID: t.TenantID, Name: t.Name, Category: string(t.Category),
		Channels: messageTemplateChannels(t.Channels), Subject: t.Subject, Body: t.Body,
		Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toMessageTemplateDomain(m messageTemplateModel) mcDomain.MessageTemplate {
	return mcDomain.MessageTemplate{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, Category: mcDomain.Category(m.Category),
		Channels: []messaging.Channel(m.Channels), Subject: m.Subject, Body: m.Body,
		Enabled: m.Enabled, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
