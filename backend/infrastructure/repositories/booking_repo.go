package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type bookingModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	PurchaseOrderID string `gorm:"not null;type:uuid"`
	Status          string `gorm:"not null;default:in_progress"`
	StartedAt       time.Time
	PausedAt        *time.Time
	CompletedAt     *time.Time
	CompletedBy     string
}

func (bookingModel) TableName() string { return "bookings" }

type bookingScanModel struct {
	ID               string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID         string  `gorm:"not null;type:uuid"`
	BookingID        string  `gorm:"not null;type:uuid"`
	SKUID            string  `gorm:"column:sku_id;not null;type:uuid"`
	BatchID          *string `gorm:"type:uuid"`
	LotNumber        string
	ExpiryDate       *time.Time
	ActionType       string `gorm:"not null;default:receive"`
	Quantity         int64
	WriteOffQuantity int64
	Reason           string
	ScannedAt        time.Time
	ScannedBy        string
	ReplayedAt       *time.Time
}

func (bookingScanModel) TableName() string { return "booking_scans" }

// ── BookingRepo ─────────────────────────────────────────────────────────────

type BookingRepo struct{ db *gorm.DB }

func NewBookingRepo(db *gorm.DB) *BookingRepo { return &BookingRepo{db: db} }

func (r *BookingRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("bookingRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BookingRepo) Create(ctx context.Context, b *procurementDomain.Booking) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := bookingModel{
		ID: b.ID, TenantID: b.TenantID, PurchaseOrderID: b.PurchaseOrderID, Status: string(b.Status),
		StartedAt: b.StartedAt, PausedAt: b.PausedAt, CompletedAt: b.CompletedAt, CompletedBy: b.CompletedBy,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("bookingRepo.Create: %w", err)
	}
	b.ID = m.ID
	return nil
}

func (r *BookingRepo) FindByID(ctx context.Context, id string) (*procurementDomain.Booking, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m bookingModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, procurementDomain.ErrBookingNotFound
		}
		return nil, fmt.Errorf("bookingRepo.FindByID: %w", err)
	}
	return toBookingDomain(m), nil
}

func (r *BookingRepo) FindActiveByPurchaseOrder(ctx context.Context, purchaseOrderID string) (*procurementDomain.Booking, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m bookingModel
	err = tx.Where("purchase_order_id = ? AND status IN ?", purchaseOrderID,
		[]string{string(procurementDomain.BookingStatusInProgress), string(procurementDomain.BookingStatusPaused)}).
		Order("started_at desc").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("bookingRepo.FindActiveByPurchaseOrder: %w", err)
	}
	return toBookingDomain(m), nil
}

func (r *BookingRepo) Update(ctx context.Context, b *procurementDomain.Booking) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := bookingModel{
		ID: b.ID, TenantID: b.TenantID, PurchaseOrderID: b.PurchaseOrderID, Status: string(b.Status),
		StartedAt: b.StartedAt, PausedAt: b.PausedAt, CompletedAt: b.CompletedAt, CompletedBy: b.CompletedBy,
	}
	if err := tx.Model(&bookingModel{}).Where("id = ?", b.ID).Updates(&m).Error; err != nil {
		return fmt.Errorf("bookingRepo.Update: %w", err)
	}
	return nil
}

func (r *BookingRepo) ListCompleted(ctx context.Context) ([]procurementDomain.Booking, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []bookingModel
	if err := tx.Where("status = ?", string(procurementDomain.BookingStatusCompleted)).
		Order("completed_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("bookingRepo.ListCompleted: %w", err)
	}
	out := make([]procurementDomain.Booking, len(rows))
	for i, m := range rows {
		out[i] = *toBookingDomain(m)
	}
	return out, nil
}

func toBookingDomain(m bookingModel) *procurementDomain.Booking {
	return &procurementDomain.Booking{
		ID: m.ID, TenantID: m.TenantID, PurchaseOrderID: m.PurchaseOrderID, Status: procurementDomain.BookingStatus(m.Status),
		StartedAt: m.StartedAt, PausedAt: m.PausedAt, CompletedAt: m.CompletedAt, CompletedBy: m.CompletedBy,
	}
}

// ── BookingScanRepo ───────────────────────────────────────────────────────────

type BookingScanRepo struct{ db *gorm.DB }

func NewBookingScanRepo(db *gorm.DB) *BookingScanRepo { return &BookingScanRepo{db: db} }

func (r *BookingScanRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("bookingScanRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BookingScanRepo) Create(ctx context.Context, s *procurementDomain.BookingScan) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := bookingScanModel{
		ID: s.ID, TenantID: s.TenantID, BookingID: s.BookingID, SKUID: s.SKUID, BatchID: s.BatchID,
		LotNumber: s.LotNumber, ExpiryDate: s.ExpiryDate, ActionType: string(s.ActionType),
		Quantity: s.Quantity, WriteOffQuantity: s.WriteOffQuantity, Reason: s.Reason,
		ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("bookingScanRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *BookingScanRepo) ListByBooking(ctx context.Context, bookingID string) ([]procurementDomain.BookingScan, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []bookingScanModel
	if err := tx.Where("booking_id = ?", bookingID).Order("scanned_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("bookingScanRepo.ListByBooking: %w", err)
	}
	scans := make([]procurementDomain.BookingScan, len(rows))
	for i, m := range rows {
		scans[i] = procurementDomain.BookingScan{
			ID: m.ID, TenantID: m.TenantID, BookingID: m.BookingID, SKUID: m.SKUID, BatchID: m.BatchID,
			LotNumber: m.LotNumber, ExpiryDate: m.ExpiryDate, ActionType: procurementDomain.BookingScanActionType(m.ActionType),
			Quantity: m.Quantity, WriteOffQuantity: m.WriteOffQuantity, Reason: m.Reason,
			ScannedAt: m.ScannedAt, ScannedBy: m.ScannedBy, ReplayedAt: m.ReplayedAt,
		}
	}
	return scans, nil
}

func (r *BookingScanRepo) MarkReplayed(ctx context.Context, id string, replayedAt time.Time, batchID *string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&bookingScanModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"replayed_at": replayedAt, "batch_id": batchID}).Error; err != nil {
		return fmt.Errorf("bookingScanRepo.MarkReplayed: %w", err)
	}
	return nil
}
