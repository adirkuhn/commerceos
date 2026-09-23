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

type pickingSessionModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	SalesOrderID string `gorm:"column:sales_order_id;not null;type:uuid"`
	Status       string `gorm:"not null;default:in_progress"`
	StartedAt    time.Time
	PausedAt     *time.Time
	CompletedAt  *time.Time
	CompletedBy  string
	LockedBy     *string    `gorm:"column:locked_by"`
	LockedAt     *time.Time `gorm:"column:locked_at"`
	LockedSince  *time.Time `gorm:"column:locked_since"`
}

func (pickingSessionModel) TableName() string { return "picking_sessions" }

type pickScanModel struct {
	ID                 string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID           string  `gorm:"not null;type:uuid"`
	SessionID          string  `gorm:"column:session_id;not null;type:uuid"`
	LineItemID         *string `gorm:"column:line_item_id;type:uuid"`
	SKUID              string  `gorm:"column:sku_id;not null;type:uuid"`
	ActionType         string  `gorm:"not null;default:pick"`
	Quantity           int64
	OriginalQtyCovered *int64 `gorm:"column:original_qty_covered"`
	ReturnToStock      bool   `gorm:"column:return_to_stock;not null;default:false"`
	PriceMinorUnits    *int64 `gorm:"column:price_minor_units"`
	Reason             string
	ScannedAt          time.Time
	ScannedBy          string
	ReplayedAt         *time.Time
}

func (pickScanModel) TableName() string { return "pick_scans" }

// ── PickingSessionRepo ───────────────────────────────────────────────────────

type PickingSessionRepo struct{ db *gorm.DB }

func NewPickingSessionRepo(db *gorm.DB) *PickingSessionRepo { return &PickingSessionRepo{db: db} }

func (r *PickingSessionRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pickingSessionRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PickingSessionRepo) Create(ctx context.Context, s *omsDomain.PickingSession) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := pickingSessionModel{
		ID: s.ID, TenantID: s.TenantID, SalesOrderID: s.SalesOrderID, Status: string(s.Status),
		StartedAt: s.StartedAt, PausedAt: s.PausedAt, CompletedAt: s.CompletedAt, CompletedBy: s.CompletedBy,
		LockedBy: s.LockedBy, LockedAt: s.LockedAt, LockedSince: s.LockedSince,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("pickingSessionRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *PickingSessionRepo) FindByID(ctx context.Context, id string) (*omsDomain.PickingSession, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m pickingSessionModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, omsDomain.ErrPickingSessionNotFound
		}
		return nil, fmt.Errorf("pickingSessionRepo.FindByID: %w", err)
	}
	return toPickingSessionDomain(m), nil
}

func (r *PickingSessionRepo) FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PickingSession, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m pickingSessionModel
	err = tx.Where("sales_order_id = ? AND status IN ?", salesOrderID,
		[]string{string(omsDomain.PickingSessionStatusInProgress), string(omsDomain.PickingSessionStatusPaused)}).
		Order("started_at desc").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("pickingSessionRepo.FindActiveBySalesOrder: %w", err)
	}
	return toPickingSessionDomain(m), nil
}

func (r *PickingSessionRepo) Update(ctx context.Context, s *omsDomain.PickingSession) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := pickingSessionModel{
		ID: s.ID, TenantID: s.TenantID, SalesOrderID: s.SalesOrderID, Status: string(s.Status),
		StartedAt: s.StartedAt, PausedAt: s.PausedAt, CompletedAt: s.CompletedAt, CompletedBy: s.CompletedBy,
		LockedBy: s.LockedBy, LockedAt: s.LockedAt, LockedSince: s.LockedSince,
	}
	if err := tx.Model(&pickingSessionModel{}).Where("id = ?", s.ID).Updates(&m).Error; err != nil {
		return fmt.Errorf("pickingSessionRepo.Update: %w", err)
	}
	return nil
}

// AcquireLock -- see PickingSessionRepository.AcquireLock's doc for why
// this is a map-based update rather than Update.
func (r *PickingSessionRepo) AcquireLock(ctx context.Context, id, userID string, at time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&pickingSessionModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"locked_by": userID, "locked_at": at,
			// Postgres evaluates every SET against the old row, so this sees
			// the previous holder: a new holder's lock starts now, a
			// heartbeat from the same holder keeps it.
			"locked_since": gorm.Expr("CASE WHEN locked_by IS DISTINCT FROM ? OR locked_since IS NULL THEN ? ELSE locked_since END", userID, at),
		}).Error; err != nil {
		return fmt.Errorf("pickingSessionRepo.AcquireLock: %w", err)
	}
	return nil
}

// ReleaseLock -- see PickingSessionRepository.ReleaseLock's doc.
func (r *PickingSessionRepo) ReleaseLock(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&pickingSessionModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"locked_by": nil, "locked_at": nil, "locked_since": nil}).Error; err != nil {
		return fmt.Errorf("pickingSessionRepo.ReleaseLock: %w", err)
	}
	return nil
}

func toPickingSessionDomain(m pickingSessionModel) *omsDomain.PickingSession {
	return &omsDomain.PickingSession{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, Status: omsDomain.PickingSessionStatus(m.Status),
		StartedAt: m.StartedAt, PausedAt: m.PausedAt, CompletedAt: m.CompletedAt, CompletedBy: m.CompletedBy,
		LockedBy: m.LockedBy, LockedAt: m.LockedAt, LockedSince: m.LockedSince,
	}
}

// ── PickScanRepo ─────────────────────────────────────────────────────────────

type PickScanRepo struct{ db *gorm.DB }

func NewPickScanRepo(db *gorm.DB) *PickScanRepo { return &PickScanRepo{db: db} }

func (r *PickScanRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pickScanRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *PickScanRepo) Create(ctx context.Context, s *omsDomain.PickScan) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := pickScanModel{
		ID: s.ID, TenantID: s.TenantID, SessionID: s.SessionID, LineItemID: s.LineItemID, SKUID: s.SKUID,
		ActionType: string(s.ActionType), Quantity: s.Quantity, OriginalQtyCovered: s.OriginalQtyCovered, ReturnToStock: s.ReturnToStock,
		PriceMinorUnits: s.PriceMinorUnits, Reason: s.Reason, ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("pickScanRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *PickScanRepo) ListBySession(ctx context.Context, sessionID string) ([]omsDomain.PickScan, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []pickScanModel
	if err := tx.Where("session_id = ?", sessionID).Order("scanned_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("pickScanRepo.ListBySession: %w", err)
	}
	scans := make([]omsDomain.PickScan, len(rows))
	for i, m := range rows {
		scans[i] = omsDomain.PickScan{
			ID: m.ID, TenantID: m.TenantID, SessionID: m.SessionID, LineItemID: m.LineItemID, SKUID: m.SKUID,
			ActionType: omsDomain.PickScanActionType(m.ActionType), Quantity: m.Quantity, OriginalQtyCovered: m.OriginalQtyCovered, ReturnToStock: m.ReturnToStock,
			PriceMinorUnits: m.PriceMinorUnits, Reason: m.Reason, ScannedAt: m.ScannedAt, ScannedBy: m.ScannedBy, ReplayedAt: m.ReplayedAt,
		}
	}
	return scans, nil
}

func (r *PickScanRepo) MarkReplayed(ctx context.Context, id string, replayedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&pickScanModel{}).Where("id = ?", id).
		Updates(map[string]interface{}{"replayed_at": replayedAt}).Error; err != nil {
		return fmt.Errorf("pickScanRepo.MarkReplayed: %w", err)
	}
	return nil
}

func (r *PickScanRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&pickScanModel{}).Error; err != nil {
		return fmt.Errorf("pickScanRepo.Delete: %w", err)
	}
	return nil
}

// ListOpenBySalesOrders -- see PickingSessionRepository.ListOpenBySalesOrders.
func (r *PickingSessionRepo) ListOpenBySalesOrders(ctx context.Context, salesOrderIDs []string) ([]omsDomain.PickingSession, error) {
	if len(salesOrderIDs) == 0 {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var models []pickingSessionModel
	if err := tx.Where("sales_order_id IN ? AND status IN ?", salesOrderIDs,
		[]string{string(omsDomain.PickingSessionStatusInProgress), string(omsDomain.PickingSessionStatusPaused)}).
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("pickingSessionRepo.ListOpenBySalesOrders: %w", err)
	}
	out := make([]omsDomain.PickingSession, len(models))
	for i, m := range models {
		out[i] = *toPickingSessionDomain(m)
	}
	return out, nil
}
