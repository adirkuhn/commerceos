package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type scaleCodeAliasModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	SKUID     string `gorm:"column:sku_id;not null;type:uuid"`
	Code      string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (scaleCodeAliasModel) TableName() string { return "scale_code_aliases" }

type ScaleCodeAliasRepo struct{ db *gorm.DB }

func NewScaleCodeAliasRepo(db *gorm.DB) *ScaleCodeAliasRepo { return &ScaleCodeAliasRepo{db: db} }

func (r *ScaleCodeAliasRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("scaleCodeAliasRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ScaleCodeAliasRepo) Create(ctx context.Context, a *pmsDomain.ScaleCodeAlias) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := scaleCodeAliasModel{
		ID: a.ID, TenantID: a.TenantID, SKUID: a.SKUID, Code: a.Code,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "scale_code_aliases_tenant_id_code_key" {
			return pmsDomain.ErrDuplicateScaleCodeAlias
		}
		return fmt.Errorf("scaleCodeAliasRepo.Create: %w", err)
	}
	a.ID = m.ID
	return nil
}

func (r *ScaleCodeAliasRepo) FindByCode(ctx context.Context, code string) (*pmsDomain.ScaleCodeAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m scaleCodeAliasModel
	if err := tx.Where("code = ?", code).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrScaleCodeAliasNotFound
		}
		return nil, fmt.Errorf("scaleCodeAliasRepo.FindByCode: %w", err)
	}
	return toScaleCodeAliasDomain(m), nil
}

func (r *ScaleCodeAliasRepo) ListBySKU(ctx context.Context, skuID string) ([]pmsDomain.ScaleCodeAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []scaleCodeAliasModel
	if err := tx.Where("sku_id = ?", skuID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("scaleCodeAliasRepo.ListBySKU: %w", err)
	}
	aliases := make([]pmsDomain.ScaleCodeAlias, len(rows))
	for i, m := range rows {
		aliases[i] = *toScaleCodeAliasDomain(m)
	}
	return aliases, nil
}

func (r *ScaleCodeAliasRepo) ListBySKUIDs(ctx context.Context, skuIDs []string) ([]pmsDomain.ScaleCodeAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []scaleCodeAliasModel
	if err := tx.Where("sku_id IN ?", skuIDs).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("scaleCodeAliasRepo.ListBySKUIDs: %w", err)
	}
	aliases := make([]pmsDomain.ScaleCodeAlias, len(rows))
	for i, m := range rows {
		aliases[i] = *toScaleCodeAliasDomain(m)
	}
	return aliases, nil
}

func (r *ScaleCodeAliasRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&scaleCodeAliasModel{})
	if res.Error != nil {
		return fmt.Errorf("scaleCodeAliasRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return pmsDomain.ErrScaleCodeAliasNotFound
	}
	return nil
}

func toScaleCodeAliasDomain(m scaleCodeAliasModel) *pmsDomain.ScaleCodeAlias {
	return &pmsDomain.ScaleCodeAlias{
		ID: m.ID, TenantID: m.TenantID, SKUID: m.SKUID, Code: m.Code,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
