package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// SupplierIssueStateRepo stores procurementDomain.SupplierIssueState rows.
type SupplierIssueStateRepo struct{ db *gorm.DB }

func NewSupplierIssueStateRepo(db *gorm.DB) *SupplierIssueStateRepo {
	return &SupplierIssueStateRepo{db: db}
}

var _ procurementDomain.SupplierIssueStateRepository = (*SupplierIssueStateRepo)(nil)

type supplierIssueStateModel struct {
	TenantID        string    `gorm:"column:tenant_id;primaryKey"`
	PurchaseOrderID string    `gorm:"column:purchase_order_id"`
	Source          string    `gorm:"column:source;primaryKey"`
	SourceID        string    `gorm:"column:source_id;primaryKey"`
	Status          string    `gorm:"column:status"`
	Note            string    `gorm:"column:note"`
	UpdatedBy       string    `gorm:"column:updated_by"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime:false"`
}

func (supplierIssueStateModel) TableName() string { return "purchase_order_supplier_issues" }

func (r *SupplierIssueStateRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("supplierIssueStateRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SupplierIssueStateRepo) ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]procurementDomain.SupplierIssueState, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []supplierIssueStateModel
	if err := tx.Where("purchase_order_id = ?", purchaseOrderID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("supplierIssueStateRepo.ListByPurchaseOrder: %w", err)
	}
	out := make([]procurementDomain.SupplierIssueState, len(rows))
	for i, m := range rows {
		out[i] = procurementDomain.SupplierIssueState{
			TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID,
			Ref:    procurementDomain.SupplierIssueRef{Source: procurementDomain.SupplierIssueSource(m.Source), SourceID: m.SourceID},
			Status: procurementDomain.SupplierIssueStatus(m.Status), Note: m.Note, UpdatedBy: m.UpdatedBy, UpdatedAt: m.UpdatedAt,
		}
	}
	return out, nil
}

func (r *SupplierIssueStateRepo) Upsert(ctx context.Context, s *procurementDomain.SupplierIssueState) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := supplierIssueStateModel{
		TenantID: s.TenantID, PurchaseOrderID: s.PurchaseOrderID, Source: string(s.Ref.Source), SourceID: s.Ref.SourceID,
		Status: string(s.Status), Note: s.Note, UpdatedBy: s.UpdatedBy, UpdatedAt: s.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "source"}, {Name: "source_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "note", "updated_by", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("supplierIssueStateRepo.Upsert: %w", err)
	}
	return nil
}
