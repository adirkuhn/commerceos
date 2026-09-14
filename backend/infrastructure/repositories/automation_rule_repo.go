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

// automationRuleConditions is a zero-dependency []mcDomain.Condition <-> jsonb
// Scanner/Valuer, same convention as pms_repo.go's stringSlice.
type automationRuleConditions []mcDomain.Condition

func (c automationRuleConditions) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]mcDomain.Condition(c))
	return string(b), err
}

func (c *automationRuleConditions) Scan(value any) error {
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
		return fmt.Errorf("automationRuleConditions.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]mcDomain.Condition)(c))
}

type automationRuleModel struct {
	ID              string                   `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string                   `gorm:"column:tenant_id;not null;type:uuid"`
	Name            string                   `gorm:"column:name;not null"`
	TriggerSubject  string                   `gorm:"column:trigger_subject;not null"`
	Conditions      automationRuleConditions `gorm:"column:conditions;type:jsonb;not null"`
	TemplateID      string                   `gorm:"column:template_id;not null;type:uuid"`
	ChannelOverride *string                  `gorm:"column:channel_override"`
	Enabled         bool                     `gorm:"column:enabled;not null"`
	CreatedAt       time.Time                `gorm:"column:created_at"`
	UpdatedAt       time.Time                `gorm:"column:updated_at"`
}

func (automationRuleModel) TableName() string { return "automation_rules" }

type AutomationRuleRepo struct{ db *gorm.DB }

func NewAutomationRuleRepo(db *gorm.DB) *AutomationRuleRepo { return &AutomationRuleRepo{db: db} }

func (r *AutomationRuleRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("automationRuleRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *AutomationRuleRepo) Create(ctx context.Context, rule *mcDomain.AutomationRule) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toAutomationRuleModel(*rule)
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("automationRuleRepo.Create: %w", err)
	}
	rule.ID = m.ID
	return nil
}

func (r *AutomationRuleRepo) FindByID(ctx context.Context, id string) (*mcDomain.AutomationRule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m automationRuleModel
	err = tx.Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, mcDomain.ErrRuleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("automationRuleRepo.FindByID: %w", err)
	}
	out := toAutomationRuleDomain(m)
	return &out, nil
}

func (r *AutomationRuleRepo) ListByTenant(ctx context.Context) ([]mcDomain.AutomationRule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []automationRuleModel
	if err := tx.Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("automationRuleRepo.ListByTenant: %w", err)
	}
	out := make([]mcDomain.AutomationRule, len(rows))
	for i, row := range rows {
		out[i] = toAutomationRuleDomain(row)
	}
	return out, nil
}

func (r *AutomationRuleRepo) ListActiveBySubject(ctx context.Context, subject string) ([]mcDomain.AutomationRule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []automationRuleModel
	if err := tx.Where("trigger_subject = ? AND enabled = true", subject).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("automationRuleRepo.ListActiveBySubject: %w", err)
	}
	out := make([]mcDomain.AutomationRule, len(rows))
	for i, row := range rows {
		out[i] = toAutomationRuleDomain(row)
	}
	return out, nil
}

func (r *AutomationRuleRepo) Update(ctx context.Context, rule *mcDomain.AutomationRule) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toAutomationRuleModel(*rule)
	if err := tx.Model(&automationRuleModel{}).Where("id = ?", rule.ID).Updates(map[string]any{
		"name": m.Name, "trigger_subject": m.TriggerSubject, "conditions": m.Conditions,
		"template_id": m.TemplateID, "channel_override": m.ChannelOverride, "enabled": m.Enabled, "updated_at": m.UpdatedAt,
	}).Error; err != nil {
		return fmt.Errorf("automationRuleRepo.Update: %w", err)
	}
	return nil
}

func (r *AutomationRuleRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&automationRuleModel{}).Error; err != nil {
		return fmt.Errorf("automationRuleRepo.Delete: %w", err)
	}
	return nil
}

func toAutomationRuleModel(r mcDomain.AutomationRule) automationRuleModel {
	var channelOverride *string
	if r.ChannelOverride != nil {
		v := string(*r.ChannelOverride)
		channelOverride = &v
	}
	return automationRuleModel{
		ID: r.ID, TenantID: r.TenantID, Name: r.Name, TriggerSubject: r.TriggerSubject,
		Conditions: automationRuleConditions(r.Conditions), TemplateID: r.TemplateID,
		ChannelOverride: channelOverride, Enabled: r.Enabled, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toAutomationRuleDomain(m automationRuleModel) mcDomain.AutomationRule {
	var channelOverride *messaging.Channel
	if m.ChannelOverride != nil {
		v := messaging.Channel(*m.ChannelOverride)
		channelOverride = &v
	}
	return mcDomain.AutomationRule{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, TriggerSubject: m.TriggerSubject,
		Conditions: []mcDomain.Condition(m.Conditions), TemplateID: m.TemplateID,
		ChannelOverride: channelOverride, Enabled: m.Enabled, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
