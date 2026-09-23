package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// StartOrResumePackingUseCase mirrors StartOrResumePickingUseCase -- reuses
// the order's existing in_progress packing session rather than creating a
// duplicate, and enforces the same per-order lock (see SessionLock). Picking
// is optional: an order that skipped the picking module may go straight to
// packing (see SalesOrder.CanStartPacking). An open picking session on the
// order is taken over when its lock allows (stale, the caller's own, or the
// caller is an admin) -- it's closed as abandoned, keeping its scans -- and
// otherwise blocks packing with ErrOrderPickingInProgress.
type StartOrResumePackingUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	sessionRepo    omsDomain.PackingSessionRepository
	pickingRepo    pickingSessionTakeover
	pickScanRepo   pickScanLister
}

func NewStartOrResumePackingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, sessionRepo omsDomain.PackingSessionRepository, pickingRepo pickingSessionTakeover, pickScanRepo pickScanLister) *StartOrResumePackingUseCase {
	return &StartOrResumePackingUseCase{db: db, salesOrderRepo: salesOrderRepo, sessionRepo: sessionRepo, pickingRepo: pickingRepo, pickScanRepo: pickScanRepo}
}

func (uc *StartOrResumePackingUseCase) Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) (*omsDomain.PackingSession, error) {
	var session omsDomain.PackingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if err := order.CanStartPacking(); err != nil {
			return err
		}
		now := time.Now()
		if err := abandonOpenPicking(ctx, uc.pickingRepo, uc.pickScanRepo, salesOrderID, actor, now); err != nil {
			return err
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			if !existing.Lock().CanTakeOver(actor, now) {
				return omsDomain.ErrPackingSessionLocked
			}
			if err := uc.sessionRepo.AcquireLock(ctx, existing.ID, actor.UserID, now); err != nil {
				return err
			}
			existing.ApplyLock(existing.Lock().AcquiredBy(actor.UserID, now))
			session = *existing
			return nil
		}

		session = omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: now,
		}
		session.ApplyLock(omsDomain.SessionLock{}.AcquiredBy(actor.UserID, now))
		return uc.sessionRepo.Create(ctx, &session)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePacking: %w", err)
	}
	return &session, nil
}
