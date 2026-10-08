package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type lineDiscrepancyModel struct {
	ID                     string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID               string `gorm:"not null;type:uuid"`
	SalesOrderID           string `gorm:"not null;type:uuid"`
	Kind                   string `gorm:"not null"`
	ItemKey                string `gorm:"not null"`
	SKUCode                string `gorm:"column:sku_code;not null"`
	Name                   string `gorm:"not null"`
	ChannelQuantity        int64  `gorm:"not null"`
	LocalQuantity          int64  `gorm:"not null"`
	ChannelTotalMinorUnits int64  `gorm:"not null"`
	LocalTotalMinorUnits   int64  `gorm:"not null"`
	DetectedAt             time.Time
	DismissedAt            *time.Time
	ResolvedAt             *time.Time
}

func (lineDiscrepancyModel) TableName() string { return "sales_order_line_discrepancies" }

func (m lineDiscrepancyModel) toDomain() omsDomain.LineDiscrepancy {
	return omsDomain.LineDiscrepancy{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, Kind: omsDomain.LineDiscrepancyKind(m.Kind),
		ItemKey: m.ItemKey, SKUCode: m.SKUCode, Name: m.Name,
		ChannelQuantity: m.ChannelQuantity, LocalQuantity: m.LocalQuantity,
		ChannelTotalMinorUnits: m.ChannelTotalMinorUnits, LocalTotalMinorUnits: m.LocalTotalMinorUnits,
		DetectedAt: m.DetectedAt, DismissedAt: m.DismissedAt, ResolvedAt: m.ResolvedAt,
	}
}

// LineDiscrepancyRepo implements domain/oms.LineDiscrepancyRepository.
type LineDiscrepancyRepo struct{ db *gorm.DB }

func NewLineDiscrepancyRepo(db *gorm.DB) *LineDiscrepancyRepo { return &LineDiscrepancyRepo{db: db} }

func (r *LineDiscrepancyRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("lineDiscrepancyRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *LineDiscrepancyRepo) ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.LineDiscrepancy, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []lineDiscrepancyModel
	if err := tx.Where("sales_order_id = ?", salesOrderID).Order("detected_at DESC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("lineDiscrepancyRepo.ListBySalesOrder: %w", err)
	}
	out := make([]omsDomain.LineDiscrepancy, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *LineDiscrepancyRepo) FindByID(ctx context.Context, id string) (*omsDomain.LineDiscrepancy, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var row lineDiscrepancyModel
	err = tx.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, omsDomain.ErrLineDiscrepancyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lineDiscrepancyRepo.FindByID: %w", err)
	}
	d := row.toDomain()
	return &d, nil
}

func (r *LineDiscrepancyRepo) Create(ctx context.Context, d *omsDomain.LineDiscrepancy) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := lineDiscrepancyModel{
		TenantID: d.TenantID, SalesOrderID: d.SalesOrderID, Kind: string(d.Kind), ItemKey: d.ItemKey,
		SKUCode: d.SKUCode, Name: d.Name, ChannelQuantity: d.ChannelQuantity, LocalQuantity: d.LocalQuantity,
		ChannelTotalMinorUnits: d.ChannelTotalMinorUnits, LocalTotalMinorUnits: d.LocalTotalMinorUnits,
		DetectedAt: d.DetectedAt, DismissedAt: d.DismissedAt, ResolvedAt: d.ResolvedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("lineDiscrepancyRepo.Create: %w", err)
	}
	d.ID = m.ID
	return nil
}

func (r *LineDiscrepancyRepo) Save(ctx context.Context, d *omsDomain.LineDiscrepancy) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&lineDiscrepancyModel{}).Where("id = ?", d.ID).Updates(map[string]any{
		"sku_code": d.SKUCode, "name": d.Name,
		"channel_quantity": d.ChannelQuantity, "local_quantity": d.LocalQuantity,
		"channel_total_minor_units": d.ChannelTotalMinorUnits, "local_total_minor_units": d.LocalTotalMinorUnits,
		"detected_at": d.DetectedAt, "dismissed_at": d.DismissedAt, "resolved_at": d.ResolvedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("lineDiscrepancyRepo.Save: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return omsDomain.ErrLineDiscrepancyNotFound
	}
	return nil
}

func (r *LineDiscrepancyRepo) ListOpen(ctx context.Context, page, perPage int) ([]omsDomain.OpenLineDiscrepancy, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	open := tx.Model(&lineDiscrepancyModel{}).Where("dismissed_at IS NULL AND resolved_at IS NULL")
	var total int64
	if err := open.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("lineDiscrepancyRepo.ListOpen: count: %w", err)
	}
	var rows []lineDiscrepancyModel
	if err := open.Order("detected_at DESC").Limit(perPage).Offset((page - 1) * perPage).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("lineDiscrepancyRepo.ListOpen: %w", err)
	}

	orderIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		orderIDs = append(orderIDs, row.SalesOrderID)
	}
	var orders []salesOrderModel
	if len(orderIDs) > 0 {
		if err := tx.Where("id IN ?", orderIDs).Find(&orders).Error; err != nil {
			return nil, 0, fmt.Errorf("lineDiscrepancyRepo.ListOpen: orders: %w", err)
		}
	}
	ordersByID := make(map[string]omsDomain.SalesOrder, len(orders))
	for _, o := range orders {
		ordersByID[o.ID] = *toSalesOrderDomain(o)
	}

	out := make([]omsDomain.OpenLineDiscrepancy, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.OpenLineDiscrepancy{LineDiscrepancy: row.toDomain(), Order: ordersByID[row.SalesOrderID]}
	}
	return out, total, nil
}
