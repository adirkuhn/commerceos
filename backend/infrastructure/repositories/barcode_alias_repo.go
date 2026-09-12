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

type barcodeAliasModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	SKUID        string `gorm:"column:sku_id;not null;type:uuid"`
	Code         string `gorm:"not null"`
	UnitsPerScan int64  `gorm:"not null;default:1"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (barcodeAliasModel) TableName() string { return "barcode_aliases" }

type BarcodeAliasRepo struct{ db *gorm.DB }

func NewBarcodeAliasRepo(db *gorm.DB) *BarcodeAliasRepo { return &BarcodeAliasRepo{db: db} }

func (r *BarcodeAliasRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("barcodeAliasRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BarcodeAliasRepo) Create(ctx context.Context, a *pmsDomain.BarcodeAlias) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := barcodeAliasModel{
		ID: a.ID, TenantID: a.TenantID, SKUID: a.SKUID, Code: a.Code, UnitsPerScan: a.UnitsPerScan,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "barcode_aliases_tenant_id_code_key" {
			return pmsDomain.ErrDuplicateBarcodeAliasCode
		}
		return fmt.Errorf("barcodeAliasRepo.Create: %w", err)
	}
	a.ID = m.ID
	return nil
}

func (r *BarcodeAliasRepo) FindByCode(ctx context.Context, code string) (*pmsDomain.BarcodeAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m barcodeAliasModel
	if err := tx.Where("code = ?", code).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrBarcodeAliasNotFound
		}
		return nil, fmt.Errorf("barcodeAliasRepo.FindByCode: %w", err)
	}
	return toBarcodeAliasDomain(m), nil
}

func (r *BarcodeAliasRepo) ListBySKU(ctx context.Context, skuID string) ([]pmsDomain.BarcodeAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []barcodeAliasModel
	if err := tx.Where("sku_id = ?", skuID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("barcodeAliasRepo.ListBySKU: %w", err)
	}
	aliases := make([]pmsDomain.BarcodeAlias, len(rows))
	for i, m := range rows {
		aliases[i] = *toBarcodeAliasDomain(m)
	}
	return aliases, nil
}

func (r *BarcodeAliasRepo) ListBySKUIDs(ctx context.Context, skuIDs []string) ([]pmsDomain.BarcodeAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []barcodeAliasModel
	if err := tx.Where("sku_id IN ?", skuIDs).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("barcodeAliasRepo.ListBySKUIDs: %w", err)
	}
	aliases := make([]pmsDomain.BarcodeAlias, len(rows))
	for i, m := range rows {
		aliases[i] = *toBarcodeAliasDomain(m)
	}
	return aliases, nil
}

func (r *BarcodeAliasRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&barcodeAliasModel{})
	if res.Error != nil {
		return fmt.Errorf("barcodeAliasRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return pmsDomain.ErrBarcodeAliasNotFound
	}
	return nil
}

func toBarcodeAliasDomain(m barcodeAliasModel) *pmsDomain.BarcodeAlias {
	return &pmsDomain.BarcodeAlias{
		ID: m.ID, TenantID: m.TenantID, SKUID: m.SKUID, Code: m.Code, UnitsPerScan: m.UnitsPerScan,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
