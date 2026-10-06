package scheduler

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// UpdateJobConfigUseCase is the super_admin "Scheduled Jobs" admin page's
// edit action -- lets enabled/interval_minutes be changed for an existing
// global cron job without a redeploy (previously only settable via a
// one-time migration seed row, see 085_scheduled_jobs.sql).
type UpdateJobConfigUseCase struct {
	db      *gorm.DB
	jobRepo schedulerDomain.JobRepository
	outbox  eventrecord.Enqueuer
}

func NewUpdateJobConfigUseCase(db *gorm.DB, jobRepo schedulerDomain.JobRepository, outbox eventrecord.Enqueuer) *UpdateJobConfigUseCase {
	return &UpdateJobConfigUseCase{db: db, jobRepo: jobRepo, outbox: outbox}
}

func (uc *UpdateJobConfigUseCase) Execute(ctx context.Context, jobKey string, enabled bool, intervalMinutes int) error {
	if intervalMinutes < 1 {
		return schedulerDomain.ErrInvalidIntervalMinutes
	}
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		before, err := uc.jobRepo.FindByKey(ctx, jobKey)
		if err != nil {
			return err
		}
		if err := uc.jobRepo.UpdateConfig(ctx, jobKey, enabled, intervalMinutes); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SchedulerJobUpdated, jobKey,
			eventrecord.Data{"job_key": jobKey}, eventrecord.NewChanges().
				Set("enabled", before.Enabled, enabled).
				Set("interval_minutes", before.IntervalMinutes, intervalMinutes))
	})
	if err != nil {
		return fmt.Errorf("updateJobConfig: %w", err)
	}
	return nil
}
