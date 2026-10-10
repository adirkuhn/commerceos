package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type notificationModel struct {
	ID        string                     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string                     `gorm:"not null;type:uuid"`
	Type      string                     `gorm:"not null"`
	Params    jsonbValue[map[string]any] `gorm:"column:params;type:jsonb;not null"`
	Title     string                     `gorm:"not null"`
	Message   string                     `gorm:"not null"`
	Link      string                     `gorm:"not null"`
	Read      bool                       `gorm:"not null"`
	CreatedAt time.Time
}

func (notificationModel) TableName() string { return "notifications" }

type NotificationRepo struct{ db *gorm.DB }

func NewNotificationRepo(db *gorm.DB) *NotificationRepo { return &NotificationRepo{db: db} }

func (r *NotificationRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("notifications: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *NotificationRepo) Create(ctx context.Context, n *notificationsDomain.Notification) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := notificationModel{
		TenantID: n.TenantID, Type: string(n.Type), Params: jsonbValue[map[string]any]{V: nonNilParams(n.Params)}, Title: n.Title, Message: n.Message, Link: n.Link,
		Read: n.Read, CreatedAt: n.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("notificationRepo.Create: %w", err)
	}
	n.ID = m.ID
	return nil
}

func (r *NotificationRepo) List(ctx context.Context, limit int) ([]notificationsDomain.Notification, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []notificationModel
	if err := tx.Order("created_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("notificationRepo.List: %w", err)
	}
	out := make([]notificationsDomain.Notification, len(rows))
	for i, row := range rows {
		out[i] = *toNotificationDomain(row)
	}
	return out, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&notificationModel{}).Where("id = ?", id).Update("read", true).Error; err != nil {
		return fmt.Errorf("notificationRepo.MarkRead: %w", err)
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&notificationModel{}).Where("read = false").Update("read", true).Error; err != nil {
		return fmt.Errorf("notificationRepo.MarkAllRead: %w", err)
	}
	return nil
}

func toNotificationDomain(m notificationModel) *notificationsDomain.Notification {
	return &notificationsDomain.Notification{
		ID: m.ID, TenantID: m.TenantID, Type: notificationsDomain.Type(m.Type), Params: nonNilParams(m.Params.V),
		Title: m.Title, Message: m.Message, Link: m.Link, Read: m.Read, CreatedAt: m.CreatedAt,
	}
}

func nonNilParams(p map[string]any) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	return p
}
