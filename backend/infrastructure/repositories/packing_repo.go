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

type packingSessionModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	SalesOrderID string `gorm:"column:sales_order_id;not null;type:uuid"`
	Status       string `gorm:"not null;default:in_progress"`
	StartedAt    time.Time
	CompletedAt  *time.Time
	CompletedBy  string
	LockedBy     *string    `gorm:"column:locked_by"`
	LockedAt     *time.Time `gorm:"column:locked_at"`
	LockedSince  *time.Time `gorm:"column:locked_since"`
}

func (packingSessionModel) TableName() string { return "packing_sessions" }

type packScanModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	SessionID  string `gorm:"column:session_id;not null;type:uuid"`
	LineItemID string `gorm:"column:line_item_id;not null;type:uuid"`
	SKUID      string `gorm:"column:sku_id;not null;type:uuid"`
	Quantity   int64
	ScannedAt  time.Time
	ScannedBy  string
}

func (packScanModel) TableName() string { return "pack_scans" }

// ── PackingSessionRepo ───────────────────────────────────────────────────────
// Mirrors PickingSessionRepo exactly -- see that type's doc comments.

type PackingSessionRepo struct{ db *gorm.DB }

func NewPackingSessionRepo(db *gorm.DB) *PackingSessionRepo { return &PackingSessionRepo{db: db} }

func (r *PackingSessionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("packingSessionRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PackingSessionRepo) Create(ctx context.Context, s *omsDomain.PackingSession) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := packingSessionModel{
		ID: s.ID, TenantID: s.TenantID, SalesOrderID: s.SalesOrderID, Status: string(s.Status),
		StartedAt: s.StartedAt, CompletedAt: s.CompletedAt, CompletedBy: s.CompletedBy,
		LockedBy: s.LockedBy, LockedAt: s.LockedAt, LockedSince: s.LockedSince,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("packingSessionRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *PackingSessionRepo) FindByID(ctx context.Context, id string) (*omsDomain.PackingSession, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m packingSessionModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, omsDomain.ErrPackingSessionNotFound
		}
		return nil, fmt.Errorf("packingSessionRepo.FindByID: %w", err)
	}
	return toPackingSessionDomain(m), nil
}

func (r *PackingSessionRepo) FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PackingSession, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m packingSessionModel
	err = tx.Where("sales_order_id = ? AND status = ?", salesOrderID, string(omsDomain.PackingSessionStatusInProgress)).
		Order("started_at desc").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("packingSessionRepo.FindActiveBySalesOrder: %w", err)
	}
	return toPackingSessionDomain(m), nil
}

func (r *PackingSessionRepo) Update(ctx context.Context, s *omsDomain.PackingSession) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := packingSessionModel{
		ID: s.ID, TenantID: s.TenantID, SalesOrderID: s.SalesOrderID, Status: string(s.Status),
		StartedAt: s.StartedAt, CompletedAt: s.CompletedAt, CompletedBy: s.CompletedBy,
		LockedBy: s.LockedBy, LockedAt: s.LockedAt, LockedSince: s.LockedSince,
	}
	if err := tx.Model(&packingSessionModel{}).Where("id = ?", s.ID).Updates(&m).Error; err != nil {
		return fmt.Errorf("packingSessionRepo.Update: %w", err)
	}
	return nil
}

// AcquireLock -- see PackingSessionRepository.AcquireLock's doc for why
// this is a map-based update rather than Update.
func (r *PackingSessionRepo) AcquireLock(ctx context.Context, id, userID string, at time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&packingSessionModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"locked_by": userID, "locked_at": at,
			// Postgres evaluates every SET against the old row, so this sees
			// the previous holder: a new holder's lock starts now, a
			// heartbeat from the same holder keeps it.
			"locked_since": gorm.Expr("CASE WHEN locked_by IS DISTINCT FROM ? OR locked_since IS NULL THEN ? ELSE locked_since END", userID, at),
		}).Error; err != nil {
		return fmt.Errorf("packingSessionRepo.AcquireLock: %w", err)
	}
	return nil
}

// ReleaseLock -- see PackingSessionRepository.ReleaseLock's doc.
func (r *PackingSessionRepo) ReleaseLock(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&packingSessionModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"locked_by": nil, "locked_at": nil, "locked_since": nil}).Error; err != nil {
		return fmt.Errorf("packingSessionRepo.ReleaseLock: %w", err)
	}
	return nil
}

func toPackingSessionDomain(m packingSessionModel) *omsDomain.PackingSession {
	return &omsDomain.PackingSession{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, Status: omsDomain.PackingSessionStatus(m.Status),
		StartedAt: m.StartedAt, CompletedAt: m.CompletedAt, CompletedBy: m.CompletedBy,
		LockedBy: m.LockedBy, LockedAt: m.LockedAt, LockedSince: m.LockedSince,
	}
}

// ── PackScanRepo ─────────────────────────────────────────────────────────────

type PackScanRepo struct{ db *gorm.DB }

func NewPackScanRepo(db *gorm.DB) *PackScanRepo { return &PackScanRepo{db: db} }

func (r *PackScanRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("packScanRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PackScanRepo) Create(ctx context.Context, s *omsDomain.PackScan) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := packScanModel{
		ID: s.ID, TenantID: s.TenantID, SessionID: s.SessionID, LineItemID: s.LineItemID, SKUID: s.SKUID,
		Quantity: s.Quantity, ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("packScanRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *PackScanRepo) ListBySession(ctx context.Context, sessionID string) ([]omsDomain.PackScan, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []packScanModel
	if err := tx.Where("session_id = ?", sessionID).Order("scanned_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("packScanRepo.ListBySession: %w", err)
	}
	scans := make([]omsDomain.PackScan, len(rows))
	for i, m := range rows {
		scans[i] = omsDomain.PackScan{
			ID: m.ID, TenantID: m.TenantID, SessionID: m.SessionID, LineItemID: m.LineItemID, SKUID: m.SKUID,
			Quantity: m.Quantity, ScannedAt: m.ScannedAt, ScannedBy: m.ScannedBy,
		}
	}
	return scans, nil
}

func (r *PackScanRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&packScanModel{}).Error; err != nil {
		return fmt.Errorf("packScanRepo.Delete: %w", err)
	}
	return nil
}

// ListActiveBySalesOrders -- see PackingSessionRepository.ListActiveBySalesOrders.
func (r *PackingSessionRepo) ListActiveBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]omsDomain.PackingSession, error) {
	if len(salesOrderIDs) == 0 {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var models []packingSessionModel
	if err := tx.Where("sales_order_id IN ? AND status = ?", salesOrderIDs, string(omsDomain.PackingSessionStatusInProgress)).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("packingSessionRepo.ListActiveBySalesOrders: %w", err)
	}
	out := make([]omsDomain.PackingSession, len(models))
	for i, m := range models {
		out[i] = *toPackingSessionDomain(m)
	}
	return out, nil
}
