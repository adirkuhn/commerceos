package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// ProductMergeRepo stores the audit trail of product merges
// (migration 185_product_merges.sql).
type ProductMergeRepo struct{}

func NewProductMergeRepo() *ProductMergeRepo { return &ProductMergeRepo{} }

type productMergeModel struct {
	ID                string    `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID          string    `gorm:"column:tenant_id"`
	SourceProductID   string    `gorm:"column:source_product_id"`
	SourceProductName string    `gorm:"column:source_product_name"`
	TargetProductID   string    `gorm:"column:target_product_id"`
	SKUMapping        []byte    `gorm:"column:sku_mapping;type:jsonb"`
	MergedBy          string    `gorm:"column:merged_by"`
	MergedAt          time.Time `gorm:"column:merged_at"`
}

func (productMergeModel) TableName() string { return "product_merges" }

type mergedSKUJSON struct {
	SourceSKUID   string `json:"source_sku_id"`
	SourceSKUCode string `json:"source_sku_code"`
	SourceBarcode string `json:"source_barcode"`
	TargetSKUID   string `json:"target_sku_id"`
}

func (ProductMergeRepo) Create(ctx context.Context, r *pmsDomain.ProductMergeRecord) error {
	tx, err := mergeTx(ctx)
	if err != nil {
		return err
	}
	skus := make([]mergedSKUJSON, len(r.SKUs))
	for i, s := range r.SKUs {
		skus[i] = mergedSKUJSON(s)
	}
	mapping, err := json.Marshal(skus)
	if err != nil {
		return fmt.Errorf("productMergeRepo.Create: %w", err)
	}
	m := productMergeModel{
		TenantID: r.TenantID, SourceProductID: r.SourceProductID, SourceProductName: r.SourceProductName,
		TargetProductID: r.TargetProductID, SKUMapping: mapping, MergedBy: r.MergedBy, MergedAt: r.MergedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("productMergeRepo.Create: %w", err)
	}
	r.ID = m.ID
	return nil
}

// MergeCodeAliases keeps a merged duplicate's barcode / scale code
// scanning to the existing SKU (application/pms.mergeCodeAliases).
type MergeCodeAliases struct {
	barcodes *BarcodeAliasRepo
	scales   *ScaleCodeAliasRepo
}

func NewMergeCodeAliases(barcodes *BarcodeAliasRepo, scales *ScaleCodeAliasRepo) *MergeCodeAliases {
	return &MergeCodeAliases{barcodes: barcodes, scales: scales}
}

// CodeTaken: any SKU's barcode or scale code, or any alias, already this.
func (a *MergeCodeAliases) CodeTaken(ctx context.Context, code string) (bool, error) {
	tx, err := mergeTx(ctx)
	if err != nil {
		return false, err
	}
	var taken bool
	err = tx.Raw(`SELECT EXISTS (SELECT 1 FROM skus WHERE barcode = @c OR scale_code = @c)
		OR EXISTS (SELECT 1 FROM barcode_aliases WHERE code = @c)
		OR EXISTS (SELECT 1 FROM scale_code_aliases WHERE code = @c)`, map[string]any{"c": code}).Scan(&taken).Error
	if err != nil {
		return false, fmt.Errorf("mergeCodeAliases.CodeTaken: %w", err)
	}
	return taken, nil
}

func (a *MergeCodeAliases) CreateBarcodeAlias(ctx context.Context, alias *pmsDomain.BarcodeAlias) error {
	return a.barcodes.Create(ctx, alias)
}

func (a *MergeCodeAliases) CreateScaleCodeAlias(ctx context.Context, alias *pmsDomain.ScaleCodeAlias) error {
	return a.scales.Create(ctx, alias)
}
