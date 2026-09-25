package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type webhookForwardRuleModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	Provider  string `gorm:"not null"`
	Topic     string `gorm:"not null"`
	TargetURL string `gorm:"column:target_url;not null"`
	Active    bool   `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (webhookForwardRuleModel) TableName() string { return "webhook_forward_rules" }

type WebhookForwardRuleRepo struct{ db *gorm.DB }

func NewWebhookForwardRuleRepo(db *gorm.DB) *WebhookForwardRuleRepo {
	return &WebhookForwardRuleRepo{db: db}
}

func (r *WebhookForwardRuleRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("webhookForwardRuleRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *WebhookForwardRuleRepo) Create(ctx context.Context, rule *webhooks.ForwardRule) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := webhookForwardRuleModel{
		TenantID: rule.TenantID, Provider: rule.Provider, Topic: rule.Topic,
		TargetURL: rule.TargetURL, Active: rule.Active,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("webhookForwardRuleRepo.Create: %w", err)
	}
	rule.ID, rule.CreatedAt, rule.UpdatedAt = m.ID, m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *WebhookForwardRuleRepo) List(ctx context.Context) ([]webhooks.ForwardRule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []webhookForwardRuleModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("webhookForwardRuleRepo.List: %w", err)
	}
	out := make([]webhooks.ForwardRule, len(rows))
	for i, row := range rows {
		out[i] = toForwardRuleDomain(row)
	}
	return out, nil
}

func (r *WebhookForwardRuleRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&webhookForwardRuleModel{}).Error; err != nil {
		return fmt.Errorf("webhookForwardRuleRepo.Delete: %w", err)
	}
	return nil
}

func (r *WebhookForwardRuleRepo) Match(ctx context.Context, provider, topic string) ([]webhooks.ForwardRule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []webhookForwardRuleModel
	if err := tx.Where("provider = ? AND active AND (topic = ? OR topic = '')", provider, topic).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("webhookForwardRuleRepo.Match: %w", err)
	}
	out := make([]webhooks.ForwardRule, len(rows))
	for i, row := range rows {
		out[i] = toForwardRuleDomain(row)
	}
	return out, nil
}

func toForwardRuleDomain(m webhookForwardRuleModel) webhooks.ForwardRule {
	return webhooks.ForwardRule{
		ID: m.ID, TenantID: m.TenantID, Provider: m.Provider, Topic: m.Topic,
		TargetURL: m.TargetURL, Active: m.Active, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
