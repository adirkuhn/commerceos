package procurement

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeletePurchaseOrderDocumentUseCase struct {
	db           *gorm.DB
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
	storage      pmsDomain.FileStorage
	outbox       eventrecord.Enqueuer
}

func NewDeletePurchaseOrderDocumentUseCase(
	db *gorm.DB,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
	storage pmsDomain.FileStorage,
	outbox eventrecord.Enqueuer,
) *DeletePurchaseOrderDocumentUseCase {
	return &DeletePurchaseOrderDocumentUseCase{db: db, documentRepo: documentRepo, storage: storage, outbox: outbox}
}

// Execute deletes the DB row first, then the stored file -- same order as
// pms.DeleteMediaUseCase, and for the same reason: a failure here never
// leaves a document row pointing at a file that's already gone (a broken
// link), worst case on a storage failure is an orphaned file using disk
// space, which is the safer side to fail on. DocumentParsingJob rows for
// this document cascade-delete at the DB level (ON DELETE CASCADE, see
// migrations/027_procurement_documents.sql).
func (uc *DeletePurchaseOrderDocumentUseCase) Execute(ctx context.Context, id string) error {
	var url string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		doc, err := uc.documentRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		url = doc.FileURL
		if err := uc.documentRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderDocumentDeleted, id, eventrecord.Data{
			"purchase_order_id": doc.PurchaseOrderID, "file_name": doc.FileName,
		})
	})
	if err != nil {
		return fmt.Errorf("deletePurchaseOrderDocument: %w", err)
	}

	if err := uc.storage.Delete(ctx, url); err != nil {
		return fmt.Errorf("deletePurchaseOrderDocument: %w", err)
	}
	return nil
}
