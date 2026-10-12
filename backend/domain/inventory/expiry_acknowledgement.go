package inventory

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/domain/settings"
)

// ExpiryAcknowledgement records that someone has seen a batch while it was
// in one close-to-expiry window ("mark as aware"). The batch then leaves
// that window's list and email report, and comes back once it moves into
// the next, smaller window -- which has no acknowledgement of its own.
type ExpiryAcknowledgement struct {
	TenantID       string
	BatchID        string
	WindowDays     int
	AcknowledgedBy string
	AcknowledgedAt time.Time
}

// NewExpiryAcknowledgement acknowledges batch for band. The batch must be
// in that band today: acknowledging a window it is not in would hide
// nothing now and silence a report it has not had yet.
func NewExpiryAcknowledgement(batch Batch, band settings.ExpiryBand, today time.Time, by string, at time.Time) (ExpiryAcknowledgement, error) {
	if batch.ExpiryDate == nil || !band.Contains(DaysUntil(*batch.ExpiryDate, today)) {
		return ExpiryAcknowledgement{}, ErrBatchNotInExpiryWindow
	}
	return ExpiryAcknowledgement{
		TenantID: batch.TenantID, BatchID: batch.ID, WindowDays: band.WindowDays,
		AcknowledgedBy: by, AcknowledgedAt: at,
	}, nil
}

// ExpiryAcknowledgementRepository stores acknowledgements, one per batch
// and window.
type ExpiryAcknowledgementRepository interface {
	// Create stores ack; created is false when the batch was already
	// acknowledged for that window (nothing changes).
	Create(ctx context.Context, ack ExpiryAcknowledgement) (created bool, err error)
	// Delete removes the batch's acknowledgement for windowDays; deleted is
	// false when there was none.
	Delete(ctx context.Context, batchID string, windowDays int) (deleted bool, err error)
}
