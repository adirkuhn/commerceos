package scheduler

import (
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"context"
	"fmt"
	"time"

	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// JobRunner is what each of this package's Run*UseCase types (
// RunExpiredSalePricesUseCase, RunExpiredBatchesUseCase,
// RunCostPriceRecalculationUseCase) satisfies -- narrowed to point of use
// (ISP) so RunJobNowUseCase can dispatch by job key without knowing which
// concrete use case it's holding, the same "small map of key -> runner"
// shape cmd/scheduler/dispatch.go already uses for SyncSchedule.
type JobRunner interface {
	Run(ctx context.Context) error
}

// RunJobNowUseCase is the super_admin "Scheduled Jobs" admin page's
// run-now action for a global cron job -- runs it synchronously (these are
// small, bounded sweeps, not per-connection paginated imports) and records
// the outcome the same way cmd/scheduler's periodic due-check loop does, so
// "last run" stays accurate regardless of which process actually ran it.
type RunJobNowUseCase struct {
	db      *gorm.DB
	jobRepo schedulerDomain.JobRepository
	runners map[string]JobRunner
	outbox  eventrecord.Enqueuer
}

func NewRunJobNowUseCase(db *gorm.DB, jobRepo schedulerDomain.JobRepository, runners map[string]JobRunner, outbox eventrecord.Enqueuer) *RunJobNowUseCase {
	return &RunJobNowUseCase{db: db, jobRepo: jobRepo, runners: runners, outbox: outbox}
}

func (uc *RunJobNowUseCase) Execute(ctx context.Context, jobKey string) error {
	runner, ok := uc.runners[jobKey]
	if !ok {
		return fmt.Errorf("runJobNow: %w: %s", schedulerDomain.ErrJobNotFound, jobKey)
	}

	adminCtx := tenant.WithSuperAdmin(ctx)
	var job *schedulerDomain.Job
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		job, err = uc.jobRepo.FindByKey(ctx, jobKey)
		return err
	})
	if err != nil {
		return fmt.Errorf("runJobNow: %w", err)
	}

	ranAt := time.Now()
	runErr := runner.Run(adminCtx)
	status, errMsg := "success", ""
	if runErr != nil {
		status, errMsg = "error", runErr.Error()
	}
	nextRunAt := ranAt.Add(time.Duration(job.IntervalMinutes) * time.Minute)

	recordErr := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		if err := uc.jobRepo.RecordRunResult(ctx, jobKey, ranAt, status, errMsg, nextRunAt); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SchedulerJobRunTriggered, jobKey, eventrecord.Data{
			"status": status, "error": errMsg,
		})
	})
	if recordErr != nil {
		return fmt.Errorf("runJobNow: %w", recordErr)
	}
	if runErr != nil {
		return fmt.Errorf("runJobNow: %w", runErr)
	}
	return nil
}
