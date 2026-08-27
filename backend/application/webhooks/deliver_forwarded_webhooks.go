package webhooks

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// maxForwardAttempts caps how many delivery attempts a ForwardDelivery row
// gets before it's marked permanently Failed for manual triage -- same
// "don't retry forever" reasoning as relay's maxOutboxPublishRetries, just
// a smaller budget since a third-party target being down for this long is
// a real, actionable problem, not a transient blip.
const maxForwardAttempts = 8

// forwardBatchLimit bounds one tick's work -- cmd/webhook-forwarder runs on
// a short poll interval, not a hot path, but an unbounded scan across
// every pending delivery for every tenant would still be an unnecessarily
// large single batch.
const forwardBatchLimit = 100

// HTTPPoster is the outbound HTTP call DeliverForwardedWebhooksUseCase
// needs, narrowed to point of use (ISP) so the use case is unit-testable
// without a real network call. See infrastructure/webhookforward for the
// real implementation.
type HTTPPoster interface {
	Post(ctx context.Context, targetURL string, contentType string, body []byte) (statusCode int, err error)
}

// DeliverForwardedWebhooksResult mirrors ReconcileShipmentsResult's
// partial-failure-tolerant shape: one bad delivery must not block the rest
// of the batch, and cmd/webhook-forwarder's caller needs to know whether
// anything failed without the whole run erroring out.
type DeliverForwardedWebhooksResult struct {
	Checked   int
	Delivered int
	Errors    []string
}

// DeliverForwardedWebhooksUseCase drains the ForwardDelivery queue --
// cmd/webhook-forwarder's per-tick job. Cross-tenant by nature (there's no
// single tenant driving this, unlike a SyncSchedule run): callers must pass
// a ctx already under tenant.WithSuperAdmin, the same convention
// cmd/scheduler's runDue uses for its own cross-tenant poll.
type DeliverForwardedWebhooksUseCase struct {
	db           *gorm.DB
	deliveryRepo webhooks.ForwardDeliveryRepository
	poster       HTTPPoster
}

func NewDeliverForwardedWebhooksUseCase(db *gorm.DB, deliveryRepo webhooks.ForwardDeliveryRepository, poster HTTPPoster) *DeliverForwardedWebhooksUseCase {
	return &DeliverForwardedWebhooksUseCase{db: db, deliveryRepo: deliveryRepo, poster: poster}
}

func (uc *DeliverForwardedWebhooksUseCase) Execute(ctx context.Context) (*DeliverForwardedWebhooksResult, error) {
	var due []webhooks.ForwardDelivery
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		due, err = uc.deliveryRepo.ListDue(ctx, time.Now(), forwardBatchLimit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("deliverForwardedWebhooks: list due: %w", err)
	}

	result := &DeliverForwardedWebhooksResult{Checked: len(due)}
	for _, d := range due {
		// Re-scoped to the delivery's own tenant for the write, matching
		// cmd/scheduler.runOne's convention -- bypass_rls from the caller's
		// SuperAdmin ctx would also satisfy RLS here, but this keeps every
		// row this use case ever touches stamped with its real tenant
		// identity rather than relying on bypass alone.
		rowCtx := tenant.WithTenantID(ctx, d.TenantID)

		status, postErr := uc.poster.Post(ctx, d.TargetURL, "application/json", d.Payload)
		if postErr == nil && status >= 200 && status < 300 {
			if markErr := tenant.WithTenant(rowCtx, uc.db, func(ctx context.Context) error {
				return uc.deliveryRepo.MarkDelivered(ctx, d.ID)
			}); markErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("delivery %s: mark delivered: %v", d.ID, markErr))
				continue
			}
			result.Delivered++
			continue
		}

		errMsg := fmt.Sprintf("unexpected status %d", status)
		if postErr != nil {
			errMsg = postErr.Error()
		}
		nextAttemptAt := time.Now().Add(forwardBackoff(d.Attempts + 1))
		if markErr := tenant.WithTenant(rowCtx, uc.db, func(ctx context.Context) error {
			return uc.deliveryRepo.MarkFailed(ctx, d.ID, errMsg, nextAttemptAt, maxForwardAttempts)
		}); markErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("delivery %s: mark failed: %v", d.ID, markErr))
			continue
		}
		result.Errors = append(result.Errors, fmt.Sprintf("delivery %s: %s", d.ID, errMsg))
	}
	return result, nil
}

// forwardBackoff is a simple doubling backoff (1, 2, 4, ... minutes)
// capped at 30 minutes -- attempts is the attempt number about to be
// recorded (1 on the first failure), so the very first retry is ~1 minute
// out and a persistently down target settles into a 30-minute poll instead
// of hammering it or waiting forever.
func forwardBackoff(attempts int) time.Duration {
	d := time.Duration(1<<uint(attempts)) * time.Minute
	const capDuration = 30 * time.Minute
	if d > capDuration {
		return capDuration
	}
	return d
}
