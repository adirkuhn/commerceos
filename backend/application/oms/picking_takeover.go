package oms

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// pickingSessionTakeover is what closing an order's open picking session
// needs from PickingSessionRepository.
type pickingSessionTakeover interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PickingSession, error)
	Update(ctx context.Context, s *omsDomain.PickingSession) error
	ReleaseLock(ctx context.Context, id string) error
}

type pickScanLister interface {
	ListBySession(ctx context.Context, sessionID string) ([]omsDomain.PickScan, error)
}

// abandonOpenPicking closes the order's open picking session, if any, as
// abandoned -- for when the order moves on without it (packing starts, or
// it's marked picked in one action). Refused while someone else actively
// holds it (unless actor is an admin), or while it still holds unapplied
// order changes that abandoning would drop. Its scans are kept.
func abandonOpenPicking(ctx context.Context, pickingRepo pickingSessionTakeover, scanRepo pickScanLister, outbox eventrecord.Enqueuer, salesOrderID string, actor omsDomain.LockActor, now time.Time) error {
	picking, err := pickingRepo.FindActiveBySalesOrder(ctx, salesOrderID)
	if err != nil || picking == nil {
		return err
	}
	if !picking.Lock().CanTakeOver(actor, now) {
		return omsDomain.ErrOrderPickingInProgress
	}
	scans, err := scanRepo.ListBySession(ctx, picking.ID)
	if err != nil {
		return err
	}
	if omsDomain.HasPendingPickChanges(scans) {
		return omsDomain.ErrPickingSessionHasPendingChanges
	}
	picking.Status = omsDomain.PickingSessionStatusAbandoned
	picking.CompletedAt = &now
	picking.CompletedBy = actor.UserID
	picking.ApplyLock(omsDomain.SessionLock{})
	if err := pickingRepo.Update(ctx, picking); err != nil {
		return err
	}
	if err := pickingRepo.ReleaseLock(ctx, picking.ID); err != nil {
		return err
	}
	return recordSession(ctx, outbox, events.PickingSessionAbandoned, picking.ID, salesOrderID, eventrecord.Data{"abandoned_by": actor.UserID})
}
