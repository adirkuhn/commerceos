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

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// jsonbValue stores any JSON-encodable value in a jsonb column -- same
// zero-dependency Valuer/Scanner approach as messageTemplateChannels.
type jsonbValue[T any] struct{ V T }

func (j jsonbValue[T]) Value() (driver.Value, error) {
	b, err := json.Marshal(j.V)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (j *jsonbValue[T]) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &j.V)
	case string:
		return json.Unmarshal([]byte(v), &j.V)
	case nil:
		return nil
	default:
		return fmt.Errorf("jsonbValue.Scan: unsupported type %T", value)
	}
}

type storedChannelPrefs struct {
	InApp bool `json:"in_app"`
	Email bool `json:"email"`
}

type notificationPreferencesModel struct {
	TenantID        string                                    `gorm:"primaryKey;type:uuid"`
	Categories      jsonbValue[map[string]storedChannelPrefs] `gorm:"column:categories;type:jsonb;not null"`
	EmailRecipients jsonbValue[[]string]                      `gorm:"column:email_recipients;type:jsonb;not null"`
	UpdatedAt       time.Time
}

func (notificationPreferencesModel) TableName() string { return "notification_preferences" }

type NotificationPreferencesRepo struct{ db *gorm.DB }

func NewNotificationPreferencesRepo(db *gorm.DB) *NotificationPreferencesRepo {
	return &NotificationPreferencesRepo{db: db}
}

func (r *NotificationPreferencesRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("notificationPreferencesRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *NotificationPreferencesRepo) Get(ctx context.Context) (*notificationsDomain.Preferences, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []notificationPreferencesModel
	if err := tx.Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("notificationPreferencesRepo.Get: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	m := rows[0]
	p := notificationsDomain.DefaultPreferences(m.TenantID)
	for c, prefs := range m.Categories.V {
		p.Categories[notificationsDomain.Category(c)] = notificationsDomain.ChannelPrefs{InApp: prefs.InApp, Email: prefs.Email}
	}
	if m.EmailRecipients.V != nil {
		p.EmailRecipients = m.EmailRecipients.V
	}
	p.UpdatedAt = m.UpdatedAt
	return p, nil
}

func (r *NotificationPreferencesRepo) Upsert(ctx context.Context, p *notificationsDomain.Preferences) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	cats := make(map[string]storedChannelPrefs, len(p.Categories))
	for c, prefs := range p.Categories {
		cats[string(c)] = storedChannelPrefs{InApp: prefs.InApp, Email: prefs.Email}
	}
	recipients := p.EmailRecipients
	if recipients == nil {
		recipients = []string{}
	}
	m := notificationPreferencesModel{
		TenantID:        p.TenantID,
		Categories:      jsonbValue[map[string]storedChannelPrefs]{V: cats},
		EmailRecipients: jsonbValue[[]string]{V: recipients},
		UpdatedAt:       p.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"categories", "email_recipients", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("notificationPreferencesRepo.Upsert: %w", err)
	}
	return nil
}
