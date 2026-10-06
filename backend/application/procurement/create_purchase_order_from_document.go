package procurement

import (
	"context"
	"fmt"
	"io"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

type CreatePurchaseOrderFromDocumentInput struct {
	// SupplierID is nil when uploading without picking a supplier -- the
	// resulting PO starts needs_review until one is linked or created (see
	// CreateSupplierAndAssignToPurchaseOrderUseCase).
	SupplierID   *string
	WarehouseID  *string
	CurrencyCode string
	ExpectedDate *time.Time
	FileName     string
	ContentType  string
	Data         io.Reader
}

// Narrow interfaces at point of use so this composition is unit-testable
// without a real DB -- CreatePurchaseOrderUseCase and
// AttachDocumentToPurchaseOrderUseCase both satisfy these structurally.
type createPurchaseOrderForDocumentUC interface {
	Execute(ctx context.Context, in CreatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error)
}
type attachDocumentUC interface {
	Execute(ctx context.Context, in AttachDocumentToPurchaseOrderInput) (*procurementDomain.DocumentParsingJob, error)
}

// CreatePurchaseOrderFromDocumentUseCase is the "upload a document to create
// a new PO" flow: creates a draft PO (warehouse is optional here too, same
// as manual creation -- see PurchaseOrder.WarehouseID), then attaches the
// document exactly like AttachDocumentToPurchaseOrderUseCase.
type CreatePurchaseOrderFromDocumentUseCase struct {
	createPO createPurchaseOrderForDocumentUC
	attach   attachDocumentUC
}

func NewCreatePurchaseOrderFromDocumentUseCase(createPO createPurchaseOrderForDocumentUC, attach attachDocumentUC) *CreatePurchaseOrderFromDocumentUseCase {
	return &CreatePurchaseOrderFromDocumentUseCase{createPO: createPO, attach: attach}
}

func (uc *CreatePurchaseOrderFromDocumentUseCase) Execute(ctx context.Context, in CreatePurchaseOrderFromDocumentInput) (*procurementDomain.DocumentParsingJob, error) {
	po, err := uc.createPO.Execute(ctx, CreatePurchaseOrderInput{
		SupplierID: in.SupplierID, WarehouseID: in.WarehouseID, CurrencyCode: in.CurrencyCode,
		ExpectedDate: in.ExpectedDate, Source: PurchaseOrderSourceDocument,
	})
	if err != nil {
		return nil, fmt.Errorf("createPurchaseOrderFromDocument: %w", err)
	}

	job, err := uc.attach.Execute(ctx, AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: po.ID, FileName: in.FileName, ContentType: in.ContentType, Data: in.Data,
	})
	if err != nil {
		return nil, fmt.Errorf("createPurchaseOrderFromDocument: %w", err)
	}
	return job, nil
}
