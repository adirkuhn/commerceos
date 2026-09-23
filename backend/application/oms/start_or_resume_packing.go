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
// duplicate, and enforces the same per-order lock (see
// PackingSession.LockedBy's doc). Unlike picking, packing can only start
// once the order's own Fulfilled flag is true (nothing to box up until
// picking says the order is done), and refuses to start a second session
// once the order is already Packed.
type StartOrResumePackingUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	sessionRepo    omsDomain.PackingSessionRepository
}

func NewStartOrResumePackingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, sessionRepo omsDomain.PackingSessionRepository) *StartOrResumePackingUseCase {
	return &StartOrResumePackingUseCase{db: db, salesOrderRepo: salesOrderRepo, sessionRepo: sessionRepo}
}

func (uc *StartOrResumePackingUseCase) Execute(ctx context.Context, salesOrderID, userID string) (*omsDomain.PackingSession, error) {
	var session omsDomain.PackingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if !order.Fulfilled {
			return omsDomain.ErrOrderNotYetPicked
		}
		if order.Packed {
			return omsDomain.ErrOrderAlreadyPacked
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.LockedBy != nil && *existing.LockedBy != userID &&
				existing.LockedAt != nil && time.Since(*existing.LockedAt) < omsDomain.PackingLockTTL {
				return omsDomain.ErrPackingSessionLocked
			}
			now := time.Now()
			if err := uc.sessionRepo.AcquireLock(ctx, existing.ID, userID, now); err != nil {
				return err
			}
			existing.LockedBy, existing.LockedAt = &userID, &now
			session = *existing
			return nil
		}

		now := time.Now()
		session = omsDomain.PackingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID,
			Status: omsDomain.PackingSessionStatusInProgress, StartedAt: now,
			LockedBy: &userID, LockedAt: &now,
		}
		return uc.sessionRepo.Create(ctx, &session)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePacking: %w", err)
	}
	return &session, nil
}
