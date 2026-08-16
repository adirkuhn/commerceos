package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM models ──────────────────────────────────────────────────────────────

type iamGroupModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string `gorm:"not null;type:uuid"`
	Name        string `gorm:"not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (iamGroupModel) TableName() string { return "iam_groups" }

type iamUserGroupModel struct {
	GroupID   string `gorm:"primaryKey;type:uuid"`
	UserID    string `gorm:"primaryKey;type:uuid"`
	TenantID  string `gorm:"not null;type:uuid"`
	CreatedAt time.Time
}

func (iamUserGroupModel) TableName() string { return "iam_user_groups" }

type iamGroupPermissionModel struct {
	GroupID   string `gorm:"primaryKey;type:uuid"`
	TenantID  string `gorm:"not null;type:uuid"`
	Rule      string `gorm:"primaryKey"`
	CreatedAt time.Time
}

func (iamGroupPermissionModel) TableName() string { return "iam_group_permissions" }

// ── GroupRepo ─────────────────────────────────────────────────────────────────

type GroupRepo struct{ db *gorm.DB }

func NewGroupRepo(db *gorm.DB) *GroupRepo { return &GroupRepo{db: db} }

func (r *GroupRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("iam: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *GroupRepo) Create(ctx context.Context, g *iamDomain.Group) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := iamGroupModel{
		ID:          g.ID,
		TenantID:    g.TenantID,
		Name:        g.Name,
		Description: g.Description,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("groupRepo.Create: %w", err)
	}
	g.ID = m.ID
	return nil
}

func (r *GroupRepo) FindByID(ctx context.Context, id string) (*iamDomain.Group, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m iamGroupModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, iamDomain.ErrGroupNotFound
		}
		return nil, fmt.Errorf("groupRepo.FindByID: %w", err)
	}
	return toGroupDomain(m), nil
}

func (r *GroupRepo) FindByName(ctx context.Context, name string) (*iamDomain.Group, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m iamGroupModel
	if err := tx.Where("name = ?", name).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, iamDomain.ErrGroupNotFound
		}
		return nil, fmt.Errorf("groupRepo.FindByName: %w", err)
	}
	return toGroupDomain(m), nil
}

func (r *GroupRepo) List(ctx context.Context) ([]iamDomain.Group, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []iamGroupModel
	if err := tx.Order("name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("groupRepo.List: %w", err)
	}
	groups := make([]iamDomain.Group, len(rows))
	for i, row := range rows {
		groups[i] = *toGroupDomain(row)
	}
	return groups, nil
}

func (r *GroupRepo) Update(ctx context.Context, g *iamDomain.Group) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&iamGroupModel{}).Where("id = ?", g.ID).
		Updates(map[string]any{
			"name":        g.Name,
			"description": g.Description,
			"updated_at":  g.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("groupRepo.Update: %w", err)
	}
	return nil
}

func (r *GroupRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&iamGroupModel{}).Error; err != nil {
		return fmt.Errorf("groupRepo.Delete: %w", err)
	}
	return nil
}

func toGroupDomain(m iamGroupModel) *iamDomain.Group {
	return &iamDomain.Group{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ── UserGroupRepo ─────────────────────────────────────────────────────────────

type UserGroupRepo struct{ db *gorm.DB }

func NewUserGroupRepo(db *gorm.DB) *UserGroupRepo { return &UserGroupRepo{db: db} }

func (r *UserGroupRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("iam: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *UserGroupRepo) AddMember(ctx context.Context, groupID, userID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	tenantID := tenant.IDFromContext(ctx)
	m := iamUserGroupModel{GroupID: groupID, UserID: userID, TenantID: tenantID, CreatedAt: time.Now()}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("userGroupRepo.AddMember: %w", err)
	}
	return nil
}

func (r *UserGroupRepo) RemoveMember(ctx context.Context, groupID, userID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&iamUserGroupModel{}).Error; err != nil {
		return fmt.Errorf("userGroupRepo.RemoveMember: %w", err)
	}
	return nil
}

func (r *UserGroupRepo) ListMembers(ctx context.Context, groupID string) ([]string, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var userIDs []string
	if err := tx.Model(&iamUserGroupModel{}).
		Where("group_id = ?", groupID).
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, fmt.Errorf("userGroupRepo.ListMembers: %w", err)
	}
	return userIDs, nil
}

func (r *UserGroupRepo) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := tx.Model(&iamUserGroupModel{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("userGroupRepo.IsMember: %w", err)
	}
	return count > 0, nil
}

// ── GroupPermissionRepo ───────────────────────────────────────────────────────

type GroupPermissionRepo struct{ db *gorm.DB }

func NewGroupPermissionRepo(db *gorm.DB) *GroupPermissionRepo {
	return &GroupPermissionRepo{db: db}
}

func (r *GroupPermissionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("iam: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *GroupPermissionRepo) Assign(ctx context.Context, groupID string, rule iamDomain.Rule) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	tenantID := tenant.IDFromContext(ctx)
	m := iamGroupPermissionModel{GroupID: groupID, TenantID: tenantID, Rule: string(rule), CreatedAt: time.Now()}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("groupPermissionRepo.Assign: %w", err)
	}
	return nil
}

func (r *GroupPermissionRepo) Revoke(ctx context.Context, groupID string, rule iamDomain.Rule) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("group_id = ? AND rule = ?", groupID, string(rule)).
		Delete(&iamGroupPermissionModel{}).Error; err != nil {
		return fmt.Errorf("groupPermissionRepo.Revoke: %w", err)
	}
	return nil
}

func (r *GroupPermissionRepo) ListByGroup(ctx context.Context, groupID string) ([]iamDomain.Rule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rules []string
	if err := tx.Model(&iamGroupPermissionModel{}).
		Where("group_id = ?", groupID).
		Order("rule asc").
		Pluck("rule", &rules).Error; err != nil {
		return nil, fmt.Errorf("groupPermissionRepo.ListByGroup: %w", err)
	}
	return toRules(rules), nil
}

func (r *GroupPermissionRepo) ListByUser(ctx context.Context, userID string) ([]iamDomain.Rule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rules []string
	if err := tx.Model(&iamGroupPermissionModel{}).
		Select("DISTINCT iam_group_permissions.rule").
		Joins("JOIN iam_user_groups ug ON ug.group_id = iam_group_permissions.group_id").
		Where("ug.user_id = ?", userID).
		Order("iam_group_permissions.rule asc").
		Pluck("iam_group_permissions.rule", &rules).Error; err != nil {
		return nil, fmt.Errorf("groupPermissionRepo.ListByUser: %w", err)
	}
	return toRules(rules), nil
}

func (r *GroupPermissionRepo) HasPermission(ctx context.Context, userID string, rule iamDomain.Rule) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := tx.Model(&iamGroupPermissionModel{}).
		Joins("JOIN iam_user_groups ug ON ug.group_id = iam_group_permissions.group_id").
		Where("ug.user_id = ? AND iam_group_permissions.rule = ?", userID, string(rule)).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("groupPermissionRepo.HasPermission: %w", err)
	}
	return count > 0, nil
}

func toRules(ss []string) []iamDomain.Rule {
	rules := make([]iamDomain.Rule, len(ss))
	for i, s := range ss {
		rules[i] = iamDomain.Rule(s)
	}
	return rules
}
