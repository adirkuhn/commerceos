package main

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	schedulerapp "github.com/adirkuhn/commerceos/application/scheduler"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// runDue polls for due schedules across every tenant — a system-level,
// cross-tenant query, so it runs under tenant.WithSuperAdmin (sets
// app.current_tenant = 'bypass_rls'), the same mechanism the JWT
// super_admin role uses. Never crashes the loop: a failure here or in any
// one schedule's run is logged and the next tick tries again.
func runDue(ctx context.Context, db *gorm.DB, scheduleRepo channelsDomain.SyncScheduleRepository, rs runners) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var due []channelsDomain.SyncSchedule
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		due, err = scheduleRepo.ListDue(ctx, time.Now())
		return err
	})
	if err != nil {
		log.Printf("list due schedules: %v", err)
		return
	}

	for _, s := range due {
		runOne(adminCtx, db, scheduleRepo, rs, s)
	}
}

// runOne dispatches a single due schedule under its own tenant's identity
// (SuperAdmin keeps RLS bypassed for the query itself, but IDFromContext
// must still resolve to the schedule's real TenantID so any rows the
// underlying use case creates — imported products, stock movements — are
// stamped with the correct tenant, not blank).
func runOne(adminCtx context.Context, db *gorm.DB, scheduleRepo channelsDomain.SyncScheduleRepository, rs runners, s channelsDomain.SyncSchedule) {
	runCtx := tenant.WithTenantID(adminCtx, s.TenantID)

	ranAt := time.Now()
	status := "success"
	errMsg := ""
	cursorAt, err := rs.dispatch(runCtx, s)
	if err != nil {
		status = "error"
		errMsg = err.Error()
		log.Printf("sync schedule %s (%s/%s): %v", s.ID, s.EntityType, s.Direction, err)
	}
	// cursorAt nil means the runner has no cursor of its own separate from
	// wall-clock time (every runner except orders import) -- see runner's
	// doc comment in dispatch.go.
	if cursorAt == nil {
		cursorAt = &ranAt
	}

	nextRunAt := ranAt.Add(time.Duration(s.IntervalMinutes) * time.Minute)
	err = tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return scheduleRepo.RecordRunResult(ctx, s.ID, ranAt, status, errMsg, nextRunAt, *cursorAt, status == "success")
	})
	if err != nil {
		log.Printf("record run result for schedule %s: %v", s.ID, err)
	}
}

// runDueJobs polls domain/scheduler's global cron-job registry (see
// migration 085_scheduled_jobs.sql) the same way runDue above polls
// SyncSchedule -- a system-level, cross-tenant query under SuperAdmin.
// Each due job is dispatched through the exact same RunJobNowUseCase a
// super_admin's manual "run now" on the admin page uses, so "last run"
// stays accurate and consistent regardless of which path actually
// triggered it. Never crashes the loop: a failure on one job (or listing
// due jobs at all) is logged and the next tick tries again.
func runDueJobs(ctx context.Context, db *gorm.DB, jobRepo schedulerDomain.JobRepository, runJobNowUC *schedulerapp.RunJobNowUseCase) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var due []schedulerDomain.Job
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		due, err = jobRepo.ListDue(ctx, time.Now())
		return err
	})
	if err != nil {
		log.Printf("list due scheduled jobs: %v", err)
		return
	}

	for _, j := range due {
		if err := runJobNowUC.Execute(adminCtx, j.JobKey); err != nil {
			log.Printf("run scheduled job %s: %v", j.JobKey, err)
		}
	}
}
