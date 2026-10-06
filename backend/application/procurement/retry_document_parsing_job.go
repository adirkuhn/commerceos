package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RetryDocumentParsingJobUseCase re-enqueues parsing for a document whose
// most recent job failed (a truncated/garbled AI response, a transient
// provider error, etc) -- same "create a new job, let cmd/scheduler pick
// it up" mechanism AttachDocumentToPurchaseOrderUseCase already uses, just
// without re-uploading: the document's FileURL already points at the
// stored file (RunDocumentParsingJobUseCase re-fetches it from storage on
// every run), so retrying only ever needs a fresh DocumentParsingJob row,
// never a new PurchaseOrderDocument. Deliberately a new job row rather
// than resetting the existing one back to pending -- keeps every attempt's
// error_message in its own row instead of overwriting history, same
// append-only-attempt convention RetryCount's own doc comment describes
// for automatic requeues.
type RetryDocumentParsingJobUseCase struct {
	db           *gorm.DB
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
	jobRepo      procurementDomain.DocumentParsingJobRepository
	outbox       eventrecord.Enqueuer
}

func NewRetryDocumentParsingJobUseCase(
	db *gorm.DB, documentRepo procurementDomain.PurchaseOrderDocumentRepository, jobRepo procurementDomain.DocumentParsingJobRepository,
	outbox eventrecord.Enqueuer,
) *RetryDocumentParsingJobUseCase {
	return &RetryDocumentParsingJobUseCase{db: db, documentRepo: documentRepo, jobRepo: jobRepo, outbox: outbox}
}

func (uc *RetryDocumentParsingJobUseCase) Execute(ctx context.Context, documentID string) (*procurementDomain.DocumentParsingJob, error) {
	var job procurementDomain.DocumentParsingJob
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		doc, err := uc.documentRepo.FindByID(ctx, documentID)
		if err != nil {
			return err
		}
		job = procurementDomain.DocumentParsingJob{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: doc.PurchaseOrderID, DocumentID: doc.ID,
			Status: procurementDomain.DocumentParsingJobPending, CreatedAt: time.Now(),
		}
		if err := uc.jobRepo.Create(ctx, &job); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderDocumentParsingRetried, doc.ID, eventrecord.Data{
			"purchase_order_id": doc.PurchaseOrderID, "parsing_job_id": job.ID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("retryDocumentParsingJob: %w", err)
	}
	return &job, nil
}
