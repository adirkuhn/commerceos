package deadletters

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type notifier interface {
	Execute(ctx context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error)
}

// CheckDeadLettersUseCase is the daily dead-letter alert (scheduler job
// check_dead_letters): every tenant with dead letters on either stream
// gets a notification in its own tenant -- "waiting", or "stale" once the
// oldest waited deadletter.StaleAfter. Dead letters without a tenant are
// only logged. Configurable alert channels are COM-16.
type CheckDeadLettersUseCase struct {
	store    lister
	notifier notifier
	now      func() time.Time
}

func NewCheckDeadLettersUseCase(store lister, notifier notifier, now func() time.Time) *CheckDeadLettersUseCase {
	return &CheckDeadLettersUseCase{store: store, notifier: notifier, now: now}
}

func (uc *CheckDeadLettersUseCase) Run(ctx context.Context) error {
	all, err := uc.listAll(ctx)
	if err != nil {
		return fmt.Errorf("checkDeadLetters: %w", err)
	}
	now := uc.now()
	var errs []error
	for _, b := range deadletter.Summarize(all) {
		if b.TenantID == "" {
			log.Printf("checkDeadLetters: %d dead letters without a tenant, oldest %dh", b.Count, b.AgeHours(now))
			continue
		}
		in := appnotifications.CreateNotificationInput{
			Type:   notificationsDomain.TypeDeadLettersWaiting,
			Params: map[string]any{"count": b.Count, "hours": b.AgeHours(now)},
		}
		if b.Stale(now) {
			in.Type = notificationsDomain.TypeDeadLettersStale
		}
		if _, err := uc.notifier.Execute(tenant.AsTenant(ctx, b.TenantID), in); err != nil {
			errs = append(errs, fmt.Errorf("notify tenant %s: %w", b.TenantID, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("checkDeadLetters: %w", err)
	}
	return nil
}

func (uc *CheckDeadLettersUseCase) listAll(ctx context.Context) ([]deadletter.DeadLetter, error) {
	var all []deadletter.DeadLetter
	for _, stream := range deadletter.Streams() {
		f := deadletter.Filter{Stream: stream, Limit: deadletter.MaxLimit}
		for {
			page, err := uc.store.List(ctx, f)
			if err != nil {
				return nil, fmt.Errorf("list %s: %w", stream, err)
			}
			all = append(all, page.Items...)
			if page.NextCursor == 0 {
				break
			}
			f.After = page.NextCursor
		}
	}
	return all, nil
}
