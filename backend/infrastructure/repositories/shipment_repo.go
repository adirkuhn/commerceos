package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type shipmentModel struct {
	ID                        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                  string `gorm:"not null;type:uuid"`
	SalesOrderID              string `gorm:"not null;type:uuid"`
	ConnectionID              string `gorm:"not null;type:uuid"`
	Carrier                   int
	ExternalShipmentID        *string `gorm:"column:external_shipment_id"`
	Barcode                   string
	Status                    int
	StatusMain                string
	TrackingURL               string
	LabelFormat               string
	PackageType               int
	SignatureRequired         bool
	OnlyRecipient             bool
	InsuranceAmountMinorUnits *int
	RecipientPerson           string
	RecipientCompany          string
	RecipientStreet           string
	RecipientNumber           string
	RecipientNumberSuffix     string
	RecipientPostalCode       string
	RecipientCity             string
	RecipientCC               string
	RecipientEmail            string
	RecipientPhone            string
	CancelledAt               *time.Time
	CancelNote                string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func (shipmentModel) TableName() string { return "shipments" }

type ShipmentRepo struct{ db *gorm.DB }

func NewShipmentRepo(db *gorm.DB) *ShipmentRepo { return &ShipmentRepo{db: db} }

func (r *ShipmentRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shipmentRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ShipmentRepo) Create(ctx context.Context, s *shippingDomain.Shipment) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toShipmentModel(s)
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("shipmentRepo.Create: %w", err)
	}
	s.ID = m.ID
	s.CreatedAt, s.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *ShipmentRepo) FindByID(ctx context.Context, id string) (*shippingDomain.Shipment, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shipmentModel
	err = tx.Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, shippingDomain.ErrShipmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("shipmentRepo.FindByID: %w", err)
	}
	return toShipmentDomain(m), nil
}

func (r *ShipmentRepo) FindByExternalShipmentID(ctx context.Context, externalShipmentID string) (*shippingDomain.Shipment, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shipmentModel
	err = tx.Where("external_shipment_id = ?", externalShipmentID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, shippingDomain.ErrShipmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("shipmentRepo.FindByExternalShipmentID: %w", err)
	}
	return toShipmentDomain(m), nil
}

func (r *ShipmentRepo) FindBySalesOrderID(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var ms []shipmentModel
	if err := tx.Where("sales_order_id = ?", salesOrderID).Order("created_at DESC").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("shipmentRepo.FindBySalesOrderID: %w", err)
	}
	out := make([]shippingDomain.Shipment, len(ms))
	for i, m := range ms {
		out[i] = *toShipmentDomain(m)
	}
	return out, nil
}

// FindStaleNonTerminal returns non-terminal shipments whose updated_at is
// older than olderThan. "delivered" is the only status_main value this
// codebase currently treats as terminal (see Shipment.IsTerminal's doc
// comment on why the full terminal set isn't confirmed) -- filtering here
// keeps that single source of truth in the domain layer's IsTerminal rather
// than duplicating the check in SQL, at the cost of fetching (and
// discarding) already-delivered rows older than the window; acceptable
// since this only runs on a scheduler tick, not a hot path.
func (r *ShipmentRepo) FindStaleNonTerminal(ctx context.Context, olderThan time.Time, limit int) ([]shippingDomain.Shipment, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var ms []shipmentModel
	q := tx.Where("updated_at < ? AND status_main != ?", olderThan, "delivered").Order("updated_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("shipmentRepo.FindStaleNonTerminal: %w", err)
	}
	out := make([]shippingDomain.Shipment, len(ms))
	for i, m := range ms {
		out[i] = *toShipmentDomain(m)
	}
	return out, nil
}

func (r *ShipmentRepo) UpdateStatus(ctx context.Context, id string, status int, statusMain, trackingURL, barcode string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	fields := map[string]any{"status": status, "status_main": statusMain, "tracking_url": trackingURL, "updated_at": time.Now()}
	// Empty means "the caller has no barcode to offer" -- never overwrite an
	// already-known one with that (see the interface doc).
	if barcode != "" {
		fields["barcode"] = barcode
	}
	err = tx.Model(&shipmentModel{}).Where("id = ?", id).Updates(fields).Error
	if err != nil {
		return fmt.Errorf("shipmentRepo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *ShipmentRepo) MarkCancelled(ctx context.Context, id string, cancelledAt time.Time, note string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Model(&shipmentModel{}).Where("id = ?", id).
		Updates(map[string]any{"cancelled_at": cancelledAt, "cancel_note": note, "updated_at": time.Now()}).Error
	if err != nil {
		return fmt.Errorf("shipmentRepo.MarkCancelled: %w", err)
	}
	return nil
}

func toShipmentModel(s *shippingDomain.Shipment) shipmentModel {
	var extID *string
	if s.ExternalShipmentID != "" {
		extID = &s.ExternalShipmentID
	}
	return shipmentModel{
		TenantID: s.TenantID, SalesOrderID: s.SalesOrderID, ConnectionID: s.ConnectionID,
		Carrier: int(s.Carrier), ExternalShipmentID: extID, Barcode: s.Barcode,
		Status: s.Status, StatusMain: s.StatusMain, TrackingURL: s.TrackingURL, LabelFormat: s.LabelFormat,
		PackageType: int(s.PackageType), SignatureRequired: s.SignatureRequired, OnlyRecipient: s.OnlyRecipient,
		InsuranceAmountMinorUnits: s.InsuranceAmountMinorEUR,
		RecipientPerson:           s.RecipientPerson, RecipientCompany: s.RecipientCompany,
		RecipientStreet: s.RecipientStreet, RecipientNumber: s.RecipientNumber, RecipientNumberSuffix: s.RecipientNumberSuffix,
		RecipientPostalCode: s.RecipientPostalCode, RecipientCity: s.RecipientCity, RecipientCC: s.RecipientCC,
		RecipientEmail: s.RecipientEmail, RecipientPhone: s.RecipientPhone,
		CancelledAt: s.CancelledAt, CancelNote: s.CancelNote,
	}
}

func toShipmentDomain(m shipmentModel) *shippingDomain.Shipment {
	var extID string
	if m.ExternalShipmentID != nil {
		extID = *m.ExternalShipmentID
	}
	return &shippingDomain.Shipment{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, ConnectionID: m.ConnectionID,
		Carrier: shippingDomain.Carrier(m.Carrier), ExternalShipmentID: extID, Barcode: m.Barcode,
		Status: m.Status, StatusMain: m.StatusMain, TrackingURL: m.TrackingURL, LabelFormat: m.LabelFormat,
		PackageType: shippingDomain.PackageType(m.PackageType), SignatureRequired: m.SignatureRequired, OnlyRecipient: m.OnlyRecipient,
		InsuranceAmountMinorEUR: m.InsuranceAmountMinorUnits,
		RecipientPerson:         m.RecipientPerson, RecipientCompany: m.RecipientCompany,
		RecipientStreet: m.RecipientStreet, RecipientNumber: m.RecipientNumber, RecipientNumberSuffix: m.RecipientNumberSuffix,
		RecipientPostalCode: m.RecipientPostalCode, RecipientCity: m.RecipientCity, RecipientCC: m.RecipientCC,
		RecipientEmail: m.RecipientEmail, RecipientPhone: m.RecipientPhone,
		CancelledAt: m.CancelledAt, CancelNote: m.CancelNote,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// --- ShipmentStatusEvent ---

type shipmentStatusEventModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	ShipmentID string `gorm:"not null;type:uuid"`
	Status     int
	StatusMain string
	Source     string
	OccurredAt time.Time
}

func (shipmentStatusEventModel) TableName() string { return "shipment_status_events" }

type ShipmentStatusEventRepo struct{ db *gorm.DB }

func NewShipmentStatusEventRepo(db *gorm.DB) *ShipmentStatusEventRepo {
	return &ShipmentStatusEventRepo{db: db}
}

func (r *ShipmentStatusEventRepo) Append(ctx context.Context, e *shippingDomain.ShipmentStatusEvent) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("shipmentStatusEventRepo: operation called outside a transaction")
	}
	m := shipmentStatusEventModel{
		TenantID: e.TenantID, ShipmentID: e.ShipmentID, Status: e.Status, StatusMain: e.StatusMain,
		Source: string(e.Source), OccurredAt: e.OccurredAt,
	}
	if err := tx.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("shipmentStatusEventRepo.Append: %w", err)
	}
	e.ID = m.ID
	return nil
}

func (r *ShipmentStatusEventRepo) ListByShipmentID(ctx context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shipmentStatusEventRepo: operation called outside a transaction")
	}
	var ms []shipmentStatusEventModel
	if err := tx.WithContext(ctx).Where("shipment_id = ?", shipmentID).Order("occurred_at DESC").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("shipmentStatusEventRepo.ListByShipmentID: %w", err)
	}
	out := make([]shippingDomain.ShipmentStatusEvent, len(ms))
	for i, m := range ms {
		out[i] = shippingDomain.ShipmentStatusEvent{
			ID: m.ID, TenantID: m.TenantID, ShipmentID: m.ShipmentID, Status: m.Status,
			StatusMain: m.StatusMain, Source: shippingDomain.StatusEventSource(m.Source), OccurredAt: m.OccurredAt,
		}
	}
	return out, nil
}
