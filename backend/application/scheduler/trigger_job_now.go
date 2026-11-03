package scheduler

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TriggerJobNowUseCase is the Scheduled Jobs page's "run now": it pulls the
// job's next run to now and the scheduler process runs it on its next poll
// (within a minute), through the one runner registry it owns. The API used
// to run jobs itself from a partial copy of that registry, so every job
// added later answered "scheduled job not found" (2026-11-03).
type TriggerJobNowUseCase struct {
	db      *gorm.DB
	jobRepo schedulerDomain.JobRepository
	outbox  eventrecord.Enqueuer
}

func NewTriggerJobNowUseCase(db *gorm.DB, jobRepo schedulerDomain.JobRepository, outbox eventrecord.Enqueuer) *TriggerJobNowUseCase {
	return &TriggerJobNowUseCase{db: db, jobRepo: jobRepo, outbox: outbox}
}

func (uc *TriggerJobNowUseCase) Execute(ctx context.Context, jobKey string) error {
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		if err := uc.jobRepo.TriggerNow(ctx, jobKey); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SchedulerJobRunRequested, jobKey)
	})
	if err != nil {
		return fmt.Errorf("triggerJobNow: %w", err)
	}
	return nil
}
