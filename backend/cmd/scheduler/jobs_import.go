package main

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	notificationsapp "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// recoverOrphanedImportJobs requeues (or, past maxJobRetries, fails) every
// job left "running" by a scheduler process that never got to mark it
// terminal — see the call site's comment for why this matters. Cross-tenant,
// so it runs under SuperAdmin like the rest of this file's polling
// functions. A failure here is logged, not fatal — worst case, a stuck job
// just waits for the next restart.
func recoverOrphanedImportJobs(ctx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxJobRetries, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned import jobs: %v", err)
		return
	}
	if requeued > 0 {
		log.Printf("requeued %d orphaned import job(s) left running by a previous scheduler process", requeued)
	}
	if failed > 0 {
		log.Printf("failed %d orphaned import job(s) that exhausted retries", failed)
	}
}

// runPendingImportJobs polls for pending ImportJobs across every tenant —
// same cross-tenant/SuperAdmin shape as runDue. Never crashes the loop: a
// failure here or in any one job's run is logged and the next tick tries
// again.
func runPendingImportJobs(
	ctx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository,
	runImportUC *channelsapp.RunManualWooImportUseCase, notifyUC *notificationsapp.DeliverNotificationUseCase,
) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []channelsDomain.ImportJob
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	if err != nil {
		log.Printf("list pending import jobs: %v", err)
		return
	}

	for _, job := range pending {
		runImportJob(adminCtx, db, jobRepo, runImportUC, notifyUC, job)
	}
}

// runImportJob runs a single pending job under its own tenant's identity
// (see runOne's comment for why) and always leaves it in a terminal state
// (completed/failed) plus a matching notification, even on failure.
// RunManualWooImportUseCase also bootstraps a full (stock-untouched) orders
// import the very first time this connection ever imports products — see
// its own doc comment; an orders-side failure there is folded into this
// job's own Errors list rather than failing the job, since the products
// import (the job's primary purpose) already succeeded.
func runImportJob(
	adminCtx context.Context, db *gorm.DB, jobRepo channelsDomain.ImportJobRepository,
	runImportUC *channelsapp.RunManualWooImportUseCase, notifyUC *notificationsapp.DeliverNotificationUseCase,
	job channelsDomain.ImportJob,
) {
	runCtx := tenant.WithTenantID(adminCtx, job.TenantID)

	startedAt := time.Now()
	if err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, startedAt)
	}); err != nil {
		log.Printf("mark import job %s running: %v", job.ID, err)
		return
	}

	result, err := runImportUC.Execute(runCtx, channelsapp.RunManualWooImportInput{
		ConnectionID: job.ConnectionID, DefaultWarehouseID: job.DefaultWarehouseID, DefaultCurrency: job.DefaultCurrency,
		IsFirstImport: job.IsFirstImport,
	})
	finishedAt := time.Now()

	if err != nil {
		log.Printf("import job %s: %v", job.ID, err)
		if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkFailed(ctx, job.ID, err.Error(), finishedAt)
		}); markErr != nil {
			log.Printf("mark import job %s failed: %v", job.ID, markErr)
		}
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeImportFailed, map[string]any{"error": err.Error()})
		return
	}

	products := result.Products
	errs := products.Errors
	params := map[string]any{"products": products.Imported, "already_linked": products.AlreadyLinked, "skipped": products.SkippedType}
	if result.Customers != nil {
		errs = append(errs, result.Customers.Errors...)
		params["customers"] = result.Customers.Created
	}
	if result.Orders != nil {
		errs = append(errs, result.Orders.Errors...)
		params["orders"] = result.Orders.Created
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, products.Imported, products.AlreadyLinked, products.SkippedType, errs, finishedAt)
	}); markErr != nil {
		log.Printf("mark import job %s completed: %v", job.ID, markErr)
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeImportCompleted, params)
}

// notifyBestEffort never aborts the caller — a notification failing to
// write shouldn't hide that the actual import job did complete or fail.
// Its text is the type's translation filled with params (see
// notificationsDomain.Type.Render).
func notifyBestEffort(ctx context.Context, notifyUC *notificationsapp.DeliverNotificationUseCase, t notificationsDomain.Type, params map[string]any) {
	if _, err := notifyUC.Execute(ctx, notificationsapp.CreateNotificationInput{Type: t, Params: params}); err != nil {
		log.Printf("notify %s: %v", t, err)
	}
}
