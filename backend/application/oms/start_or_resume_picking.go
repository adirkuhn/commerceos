package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// StartOrResumePickingUseCase reuses the order's existing in_progress/paused
// picking session rather than creating a duplicate -- same "pausing one
// order to work another is just start/resume" mechanic as
// procurement.StartOrResumeBookingUseCase. Unlike Booking, there's no
// warehouse-assignment step: SalesOrder.WarehouseID is always set.
type StartOrResumePickingUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	sessionRepo    omsDomain.PickingSessionRepository
}

func NewStartOrResumePickingUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, sessionRepo omsDomain.PickingSessionRepository) *StartOrResumePickingUseCase {
	return &StartOrResumePickingUseCase{db: db, salesOrderRepo: salesOrderRepo, sessionRepo: sessionRepo}
}

func (uc *StartOrResumePickingUseCase) Execute(ctx context.Context, salesOrderID string) (*omsDomain.PickingSession, error) {
	var session omsDomain.PickingSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if order.Fulfilled {
			return omsDomain.ErrOrderAlreadyFulfilled
		}

		existing, err := uc.sessionRepo.FindActiveBySalesOrder(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.Status == omsDomain.PickingSessionStatusPaused {
				existing.Status = omsDomain.PickingSessionStatusInProgress
				existing.PausedAt = nil
				if err := uc.sessionRepo.Update(ctx, existing); err != nil {
					return err
				}
			}
			session = *existing
			return nil
		}

		session = omsDomain.PickingSession{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: salesOrderID,
			Status: omsDomain.PickingSessionStatusInProgress, StartedAt: time.Now(),
		}
		return uc.sessionRepo.Create(ctx, &session)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumePicking: %w", err)
	}
	return &session, nil
}
