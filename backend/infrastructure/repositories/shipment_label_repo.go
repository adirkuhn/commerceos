package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type shipmentLabelModel struct {
	ShipmentID  string `gorm:"primaryKey;type:uuid"`
	TenantID    string `gorm:"not null;type:uuid"`
	ContentType string `gorm:"not null"`
	Data        []byte `gorm:"not null"`
	CreatedAt   time.Time
}

func (shipmentLabelModel) TableName() string { return "shipment_labels" }

type ShipmentLabelRepo struct{ db *gorm.DB }

var _ shippingDomain.ShipmentLabelRepository = (*ShipmentLabelRepo)(nil)

func NewShipmentLabelRepo(db *gorm.DB) *ShipmentLabelRepo { return &ShipmentLabelRepo{db: db} }

func (r *ShipmentLabelRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shipmentLabelRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ShipmentLabelRepo) Save(ctx context.Context, l *shippingDomain.ShipmentLabel) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := shipmentLabelModel{ShipmentID: l.ShipmentID, TenantID: l.TenantID, ContentType: l.ContentType, Data: l.Data, CreatedAt: l.CreatedAt}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "shipment_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content_type", "data"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("shipmentLabelRepo.Save: %w", err)
	}
	return nil
}

func (r *ShipmentLabelRepo) FindByShipmentID(ctx context.Context, shipmentID string) (*shippingDomain.ShipmentLabel, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shipmentLabelModel
	err = tx.Where("shipment_id = ?", shipmentID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("shipmentLabelRepo.FindByShipmentID: %w", err)
	}
	return &shippingDomain.ShipmentLabel{ShipmentID: m.ShipmentID, TenantID: m.TenantID, ContentType: m.ContentType, Data: m.Data, CreatedAt: m.CreatedAt}, nil
}
