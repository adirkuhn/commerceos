package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	notificationsapp "github.com/adirkuhn/commerceos/application/notifications"
	procurementapp "github.com/adirkuhn/commerceos/application/procurement"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// recoverOrphanedDocumentParsingJobs mirrors recoverOrphanedImportJobs for
// DocumentParsingJob -- see that function's comment for the full rationale.
func recoverOrphanedDocumentParsingJobs(ctx context.Context, db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxJobRetries, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned document parsing jobs: %v", err)
		return
	}
	if requeued > 0 {
		log.Printf("requeued %d orphaned document parsing job(s) left running by a previous scheduler process", requeued)
	}
	if failed > 0 {
		log.Printf("failed %d orphaned document parsing job(s) that exhausted retries", failed)
	}
}

// runPendingDocumentParsingJobs mirrors runPendingImportJobs for
// DocumentParsingJob -- same cross-tenant/SuperAdmin poll shape.
func runPendingDocumentParsingJobs(
	ctx context.Context, db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository,
	runUC *procurementapp.RunDocumentParsingJobUseCase, notifyUC *notificationsapp.DeliverNotificationUseCase,
) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []procurementDomain.DocumentParsingJob
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	if err != nil {
		log.Printf("list pending document parsing jobs: %v", err)
		return
	}

	for _, job := range pending {
		runDocumentParsingJob(adminCtx, db, jobRepo, runUC, notifyUC, job)
	}
}

// runDocumentParsingJob runs a single pending job under its own tenant's
// identity and always leaves it in a terminal state (completed/failed) plus
// a matching notification, even on failure -- same shape as runImportJob.
func runDocumentParsingJob(
	adminCtx context.Context, db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository,
	runUC *procurementapp.RunDocumentParsingJobUseCase, notifyUC *notificationsapp.DeliverNotificationUseCase,
	job procurementDomain.DocumentParsingJob,
) {
	runCtx := tenant.WithTenantID(adminCtx, job.TenantID)

	startedAt := time.Now()
	if err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, startedAt)
	}); err != nil {
		log.Printf("mark document parsing job %s running: %v", job.ID, err)
		return
	}

	needsReview, err := runUC.Execute(runCtx, job)
	finishedAt := time.Now()

	if err != nil {
		log.Printf("document parsing job %s: %v", job.ID, err)
		if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkFailed(ctx, job.ID, err.Error(), finishedAt)
		}); markErr != nil {
			log.Printf("mark document parsing job %s failed: %v", job.ID, markErr)
		}
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeDocumentParsingFailed, "Document parsing failed", err.Error())
		return
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, finishedAt)
	}); markErr != nil {
		log.Printf("mark document parsing job %s completed: %v", job.ID, markErr)
	}

	if needsReview {
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypePurchaseOrderNeedsReview, "Purchase order needs review",
			"Some line items couldn't be matched to a product automatically — review and link them.")
		return
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeDocumentParsingCompleted, "Document parsed",
		"All line items were matched to products automatically.")
}

// recoverOrphanedReceivingParsingJobs mirrors recoverOrphanedDocumentParsingJobs
// for ReceivingParsingJob -- see recoverOrphanedImportJobs's comment for the
// full rationale.
func recoverOrphanedReceivingParsingJobs(ctx context.Context, db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository) {
	adminCtx := tenant.WithSuperAdmin(ctx)
	var requeued, failed int64
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		requeued, failed, err = jobRepo.RecoverOrphanedRunning(ctx, maxJobRetries, "interrupted: scheduler restarted while this job was running", time.Now())
		return err
	})
	if err != nil {
		log.Printf("recover orphaned receiving parsing jobs: %v", err)
		return
	}
	if requeued > 0 {
		log.Printf("requeued %d orphaned receiving parsing job(s) left running by a previous scheduler process", requeued)
	}
	if failed > 0 {
		log.Printf("failed %d orphaned receiving parsing job(s) that exhausted retries", failed)
	}
}

// runPendingReceivingParsingJobs mirrors runPendingDocumentParsingJobs for
// ReceivingParsingJob -- same cross-tenant/SuperAdmin poll shape.
func runPendingReceivingParsingJobs(
	ctx context.Context, db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository,
	runUC *procurementapp.RunReceivingParsingJobUseCase, notifyUC *notificationsapp.DeliverNotificationUseCase,
) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pending []procurementDomain.ReceivingParsingJob
	err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		var err error
		pending, err = jobRepo.ListPending(ctx)
		return err
	})
	if err != nil {
		log.Printf("list pending receiving parsing jobs: %v", err)
		return
	}

	for _, job := range pending {
		runReceivingParsingJob(adminCtx, db, jobRepo, runUC, notifyUC, job)
	}
}

// runReceivingParsingJob runs a single pending job under its own tenant's
// identity and always leaves it in a terminal state (completed/failed) plus
// a matching notification, even on failure -- same shape as runDocumentParsingJob.
func runReceivingParsingJob(
	adminCtx context.Context, db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository,
	runUC *procurementapp.RunReceivingParsingJobUseCase, notifyUC *notificationsapp.DeliverNotificationUseCase,
	job procurementDomain.ReceivingParsingJob,
) {
	runCtx := tenant.WithTenantID(adminCtx, job.TenantID)

	startedAt := time.Now()
	if err := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkRunning(ctx, job.ID, startedAt)
	}); err != nil {
		log.Printf("mark receiving parsing job %s running: %v", job.ID, err)
		return
	}

	discrepancyCount, err := runUC.Execute(runCtx, job)
	finishedAt := time.Now()

	if err != nil {
		log.Printf("receiving parsing job %s: %v", job.ID, err)
		if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
			return jobRepo.MarkFailed(ctx, job.ID, err.Error(), finishedAt)
		}); markErr != nil {
			log.Printf("mark receiving parsing job %s failed: %v", job.ID, markErr)
		}
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeReceivingParsingFailed, "Invoice parsing failed", err.Error())
		return
	}

	if markErr := tenant.WithTenant(adminCtx, db, func(ctx context.Context) error {
		return jobRepo.MarkCompleted(ctx, job.ID, finishedAt)
	}); markErr != nil {
		log.Printf("mark receiving parsing job %s completed: %v", job.ID, markErr)
	}

	if discrepancyCount > 0 {
		notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeReceivingDiscrepanciesFound, "Receiving discrepancies found",
			fmt.Sprintf("The invoice didn't fully match the purchase order (%d discrepancies) — review and reconcile.", discrepancyCount))
		return
	}
	notifyBestEffort(runCtx, notifyUC, notificationsDomain.TypeReceivingParsingCompleted, "Invoice parsed",
		"The invoice matched the purchase order exactly.")
}
