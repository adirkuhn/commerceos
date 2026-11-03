// Package deadletters lets a super-admin inspect events a consumer gave up
// on, retry one for the consumer that failed, or discard it (COM-14).
package deadletters

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Dead letters span every tenant, so only a super-admin manages them. The
// route is guarded too; this keeps the rule true for any other caller.
func requireSuperAdmin(ctx context.Context) error {
	if !tenant.IsSuperAdmin(ctx) {
		return deadletter.ErrForbidden
	}
	return nil
}

// checkAccess is the guard every use case runs before touching a stream.
func checkAccess(ctx context.Context, stream deadletter.Stream) error {
	if err := requireSuperAdmin(ctx); err != nil {
		return err
	}
	_, err := deadletter.ParseStream(string(stream))
	return err
}

type getter interface {
	Get(ctx context.Context, stream deadletter.Stream, seq uint64) (deadletter.DeadLetter, error)
}

type getDeleter interface {
	getter
	Delete(ctx context.Context, stream deadletter.Stream, seq uint64) error
}

// recordAction raises action in the dead letter's own tenant, so the
// tenant's history shows it with the super-admin as actor. A dead letter
// without a tenant has nowhere to record it and raises none.
func recordAction(ctx context.Context, db *gorm.DB, outbox eventrecord.Enqueuer, action events.ActionType, d deadletter.DeadLetter, data eventrecord.Data) error {
	if d.TenantID == "" {
		return nil
	}
	return tenant.WithTenant(tenant.AsTenant(ctx, d.TenantID), db, func(ctx context.Context) error {
		if err := eventrecord.Record(ctx, outbox, action, d.EventID, data); err != nil {
			return fmt.Errorf("record %s: %w", action.Subject, err)
		}
		return nil
	})
}

func auditData(d deadletter.DeadLetter) eventrecord.Data {
	return eventrecord.Data{"stream": string(d.Stream), "sequence": d.Sequence, "subject": d.Subject, "consumer": d.Consumer}
}
