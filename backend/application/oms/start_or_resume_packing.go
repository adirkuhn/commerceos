package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// StartOrResumePackingUseCase opens the order's packing session (reusing an
// in_progress one rather than creating a duplicate). Opening only views --
// it never takes the lock; the first write does (see claimPackingLock). Picking
// is optional: an order that skipped the picking module may go straight to
// packing (see SalesOrder.CanStartPacking). An open picking session on the
// order is taken over when its lock allows (stale, the caller's own, or the
// caller is an admin) -- it's closed as abandoned, keeping its scans -- and
// otherwise blocks packing with ErrOrderPickingInProgress.
type StartOrResumePackingUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	origins        originFinder
	sessionRepo    omsDomain.PackingSessionRepository
	pickingRepo    pickingSessionTakeover
	pickScanRepo   pickScanLister
}

func NewStartOrResumePackingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, origins originFinder, sessionRepo omsDomain.PackingSessionRepository, pickingRepo pickingSessionTakeover, pickScanRepo pickScanLister, outbox eventrecord.Enqueuer) *StartOrResumePackingUseCase {
	return &StartOrResumePackingUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, origins: origins, sessionRepo: sessionRepo, pickingRepo: pickingRepo, pickScanRepo: pickScanRepo}
}

func (uc *StartOrResumePackingUseCase) Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) (*omsDomain.PackingSession, error) {
	var session omsDomain.PackingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		source, err := sourceOf(ctx, uc.origins, salesOrderID)
		if err != nil {
			return err
		}
		if err := order.CanStartPacking(source); err != nil {
			return err
		}
		now := time.Now()
		if err := abandonOpenPicking(ctx, uc.pickingRepo, uc.pickScanRepo, uc.outbox, salesOrderID, actor, now); err != nil {
			return err
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		// Opening only views -- the lock is taken by the first write (see
		// claimPackingLock), so browsing orders never locks them. A
		// session someone else holds comes back with their lock, for the
		// screen to show read-only.
		if existing != nil {
			session = *existing
			return nil
		}

		session = omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: now,
		}
		if err := uc.sessionRepo.Create(ctx, &session); err != nil {
			return err
		}
		return recordSession(ctx, uc.outbox, events.PackingSessionStarted, session.ID, salesOrderID, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePacking: %w", err)
	}
	return &session, nil
}
