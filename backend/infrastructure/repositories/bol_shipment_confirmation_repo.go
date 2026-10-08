package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type bolShipmentConfirmationModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	SalesOrderID    string `gorm:"not null;type:uuid"`
	ConnectionID    string `gorm:"not null;type:uuid"`
	ShipmentID      string `gorm:"not null;type:uuid"`
	ExternalOrderID string `gorm:"not null"`
	ProcessStatusID string `gorm:"not null"`
	TransporterCode string `gorm:"not null"`
	TrackAndTrace   string `gorm:"not null"`
	Status          string `gorm:"not null"`
	ErrorMessage    string `gorm:"not null"`
	CreatedAt       time.Time
	ResolvedAt      *time.Time
}

func (bolShipmentConfirmationModel) TableName() string { return "bol_shipment_confirmations" }

func (m bolShipmentConfirmationModel) toDomain() channelsDomain.BolShipmentConfirmation {
	return channelsDomain.BolShipmentConfirmation{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, ConnectionID: m.ConnectionID,
		ShipmentID: m.ShipmentID, ExternalOrderID: m.ExternalOrderID, ProcessStatusID: m.ProcessStatusID,
		TransporterCode: m.TransporterCode, TrackAndTrace: m.TrackAndTrace,
		Status: channelsDomain.BolConfirmationStatus(m.Status), ErrorMessage: m.ErrorMessage,
		CreatedAt: m.CreatedAt, ResolvedAt: m.ResolvedAt,
	}
}

// BolShipmentConfirmationRepo implements
// domain/channels.BolShipmentConfirmationRepository.
type BolShipmentConfirmationRepo struct{ db *gorm.DB }

func NewBolShipmentConfirmationRepo(db *gorm.DB) *BolShipmentConfirmationRepo {
	return &BolShipmentConfirmationRepo{db: db}
}

func (r *BolShipmentConfirmationRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("bolShipmentConfirmationRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BolShipmentConfirmationRepo) Create(ctx context.Context, c *channelsDomain.BolShipmentConfirmation) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	createdAt := c.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	m := bolShipmentConfirmationModel{
		TenantID: c.TenantID, SalesOrderID: c.SalesOrderID, ConnectionID: c.ConnectionID, ShipmentID: c.ShipmentID,
		ExternalOrderID: c.ExternalOrderID, ProcessStatusID: c.ProcessStatusID, TransporterCode: c.TransporterCode,
		TrackAndTrace: c.TrackAndTrace, Status: string(c.Status), ErrorMessage: c.ErrorMessage,
		CreatedAt: createdAt, ResolvedAt: c.ResolvedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("bolShipmentConfirmationRepo.Create: %w", err)
	}
	c.ID = m.ID
	c.CreatedAt = m.CreatedAt
	return nil
}

func (r *BolShipmentConfirmationRepo) ListBySalesOrder(ctx context.Context, salesOrderID string) ([]channelsDomain.BolShipmentConfirmation, error) {
	return r.list(ctx, "bolShipmentConfirmationRepo.ListBySalesOrder", func(q *gorm.DB) *gorm.DB {
		return q.Where("sales_order_id = ?", salesOrderID).Order("created_at DESC")
	})
}

func (r *BolShipmentConfirmationRepo) ListPending(ctx context.Context, limit int) ([]channelsDomain.BolShipmentConfirmation, error) {
	return r.list(ctx, "bolShipmentConfirmationRepo.ListPending", func(q *gorm.DB) *gorm.DB {
		return q.Where("status = ?", string(channelsDomain.BolConfirmationPending)).Order("created_at ASC").Limit(limit)
	})
}

func (r *BolShipmentConfirmationRepo) list(ctx context.Context, op string, scope func(*gorm.DB) *gorm.DB) ([]channelsDomain.BolShipmentConfirmation, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []bolShipmentConfirmationModel
	if err := scope(tx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	out := make([]channelsDomain.BolShipmentConfirmation, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *BolShipmentConfirmationRepo) SaveResolution(ctx context.Context, c *channelsDomain.BolShipmentConfirmation) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&bolShipmentConfirmationModel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"status": string(c.Status), "error_message": c.ErrorMessage, "resolved_at": c.ResolvedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("bolShipmentConfirmationRepo.SaveResolution: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return channelsDomain.ErrBolConfirmationNotFound
	}
	return nil
}
