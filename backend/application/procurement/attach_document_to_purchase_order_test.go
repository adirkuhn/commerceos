package procurement_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"strings"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachDocumentToPurchaseOrder_savesDocumentAndEnqueuesJob(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	documentRepo := &stubDocumentRepo{}
	lineItemRepo := &stubLineItemRepo{}
	jobRepo := &stubParsingJobRepo{}
	storage := &stubFileStorage{}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, lineItemRepo, jobRepo, storage, &eventrecordtest.Outbox{})

	job, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-1", FileName: "order.csv", ContentType: "text/csv", Data: strings.NewReader("sku,qty\nA,1"),
	})

	require.NoError(t, err)
	require.Len(t, documentRepo.created, 1)
	assert.Equal(t, "po-1", documentRepo.created[0].PurchaseOrderID)
	assert.Equal(t, "order.csv", documentRepo.created[0].FileName)
	require.Len(t, jobRepo.created, 1)
	assert.Equal(t, "document-1", jobRepo.created[0].DocumentID)
	assert.Equal(t, procurementDomain.DocumentParsingJobPending, job.Status)
	assert.Equal(t, "sku,qty\nA,1", string(storage.savedData))
}

func TestAttachDocumentToPurchaseOrder_flagsDocumentAsDuplicate_onSamePOReupload(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	documentRepo := &stubDocumentRepo{duplicateCandidates: []procurementDomain.PurchaseOrderDocument{
		{ID: "document-0", PurchaseOrderID: "po-1"},
	}}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, &stubLineItemRepo{}, &stubParsingJobRepo{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-1", FileName: "invoice.pdf", ContentType: "application/pdf", Data: strings.NewReader("%PDF-1.4 fake bytes"),
	})

	require.NoError(t, err)
	require.Len(t, documentRepo.created, 1)
	assert.True(t, documentRepo.created[0].IsPotentialDuplicate)
	assert.NotEmpty(t, documentRepo.created[0].ContentHash)
	assert.Equal(t, "po-1", documentRepo.gotDuplicatePOID)
	assert.Empty(t, poRepo.duplicatesSet, "a re-upload onto the same PO is a duplicate document, not a duplicate PO")
}

func TestAttachDocumentToPurchaseOrder_marksPurchaseOrderAsDuplicate_whenMatchIsOnADifferentPO(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-2"}}
	documentRepo := &stubDocumentRepo{duplicateCandidates: []procurementDomain.PurchaseOrderDocument{
		{ID: "document-0", PurchaseOrderID: "po-1"},
	}}
	// The matched PO (po-1) has no line items yet, so this falls back to a
	// normal parse -- the "copy instead of parse" path is covered in its
	// own tests below.
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, &stubLineItemRepo{}, &stubParsingJobRepo{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-2", FileName: "invoice.pdf", ContentType: "application/pdf", Data: strings.NewReader("%PDF-1.4 fake bytes"),
	})

	require.NoError(t, err)
	assert.True(t, documentRepo.created[0].IsPotentialDuplicate)
	require.Len(t, poRepo.duplicatesSet, 1)
	assert.Equal(t, "po-2", poRepo.duplicatesSet[0].id)
	assert.Equal(t, "po-1", poRepo.duplicatesSet[0].duplicateOfPurchaseOrderID)
}

// TestAttachDocumentToPurchaseOrder_copiesFromDuplicateInsteadOfParsing is
// the actual fix for "detect duplicate before parsing": when the matched
// PO already has line items from its own earlier parse, this document is
// known to be the same content -- copy the line items (and supplier, since
// this PO doesn't have one yet) directly instead of spending an AI/
// deterministic parse re-extracting the same thing, and the job comes back
// already Completed instead of Pending.
func TestAttachDocumentToPurchaseOrder_copiesFromDuplicateInsteadOfParsing(t *testing.T) {
	db := newMockGormDBN(t, 2)
	sourceSupplierID := "supplier-1"
	poRepo := &stubPurchaseOrderRepo{
		byID: map[string]*procurementDomain.PurchaseOrder{
			"po-2": {ID: "po-2", Status: procurementDomain.PurchaseOrderStatusNeedsReview},
			"po-1": {ID: "po-1", SupplierID: &sourceSupplierID, Status: procurementDomain.PurchaseOrderStatusDraft},
		},
		findResult: &procurementDomain.PurchaseOrder{ID: "po-2", Status: procurementDomain.PurchaseOrderStatusNeedsReview},
	}
	skuID := "sku-1"
	documentRepo := &stubDocumentRepo{duplicateCandidates: []procurementDomain.PurchaseOrderDocument{
		{ID: "document-0", PurchaseOrderID: "po-1"},
	}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-item-src", PurchaseOrderID: "po-1", SKUID: &skuID, RawSKU: "WID-1", RawDescription: "Widget", Quantity: 3, UnitCostMinorUnits: 500},
	}}
	jobRepo := &stubParsingJobRepo{}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, lineItemRepo, jobRepo, &stubFileStorage{}, &eventrecordtest.Outbox{})

	job, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-2", FileName: "invoice.pdf", ContentType: "application/pdf", Data: strings.NewReader("%PDF-1.4 fake bytes"),
	})

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.DocumentParsingJobCompleted, job.Status, "no parse needed -- the line items were copied, not extracted")
	require.NotNil(t, job.FinishedAt)

	require.Len(t, lineItemRepo.created, 1)
	assert.Equal(t, "po-2", lineItemRepo.created[0].PurchaseOrderID)
	assert.Equal(t, "WID-1", lineItemRepo.created[0].RawSKU)
	assert.Equal(t, "Widget", lineItemRepo.created[0].RawDescription)
	assert.Equal(t, int64(3), lineItemRepo.created[0].Quantity)
	require.NotNil(t, lineItemRepo.created[0].SKUID)
	assert.Equal(t, skuID, *lineItemRepo.created[0].SKUID)

	require.Len(t, poRepo.updated, 1, "supplier copied + every item already SKU-matched -- reverts straight out of needs_review")
	require.NotNil(t, poRepo.updated[0].SupplierID)
	assert.Equal(t, sourceSupplierID, *poRepo.updated[0].SupplierID)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusDraft, poRepo.updated[0].Status)
}

func TestAttachDocumentToPurchaseOrder_doesNotOverwriteSupplier_whenThisPOAlreadyHasOne(t *testing.T) {
	db := newMockGormDBN(t, 2)
	existingSupplierID := "supplier-existing"
	sourceSupplierID := "supplier-source"
	poRepo := &stubPurchaseOrderRepo{
		byID: map[string]*procurementDomain.PurchaseOrder{
			"po-2": {ID: "po-2", SupplierID: &existingSupplierID, Status: procurementDomain.PurchaseOrderStatusDraft},
			"po-1": {ID: "po-1", SupplierID: &sourceSupplierID, Status: procurementDomain.PurchaseOrderStatusDraft},
		},
		findResult: &procurementDomain.PurchaseOrder{ID: "po-2", SupplierID: &existingSupplierID, Status: procurementDomain.PurchaseOrderStatusDraft},
	}
	documentRepo := &stubDocumentRepo{duplicateCandidates: []procurementDomain.PurchaseOrderDocument{
		{ID: "document-0", PurchaseOrderID: "po-1"},
	}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-item-src", PurchaseOrderID: "po-1", RawSKU: "WID-1", RawDescription: "Widget", Quantity: 1},
	}}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, lineItemRepo, &stubParsingJobRepo{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-2", FileName: "invoice.pdf", ContentType: "application/pdf", Data: strings.NewReader("%PDF-1.4 fake bytes"),
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1, "items still copy even when the supplier itself doesn't need to")
	// Already draft with its own supplier -- nothing left to revert, so no
	// Update call at all is also an acceptable outcome; if one did happen,
	// it must not have clobbered the existing supplier.
	for _, updated := range poRepo.updated {
		require.NotNil(t, updated.SupplierID)
		assert.Equal(t, existingSupplierID, *updated.SupplierID)
	}
}

func TestAttachDocumentToPurchaseOrder_notFlaggedAsDuplicate_whenNoMatchFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	documentRepo := &stubDocumentRepo{}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, &stubLineItemRepo{}, &stubParsingJobRepo{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-1", FileName: "invoice.pdf", ContentType: "application/pdf", Data: strings.NewReader("%PDF-1.4 fake bytes"),
	})

	require.NoError(t, err)
	require.Len(t, documentRepo.created, 1)
	assert.False(t, documentRepo.created[0].IsPotentialDuplicate)
	assert.Empty(t, poRepo.duplicatesSet)
}

func TestAttachDocumentToPurchaseOrder_propagatesPurchaseOrderNotFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, &stubDocumentRepo{}, &stubLineItemRepo{}, &stubParsingJobRepo{}, &stubFileStorage{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "missing", FileName: "order.csv", ContentType: "text/csv", Data: strings.NewReader(""),
	})

	assert.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

func TestAttachDocumentToPurchaseOrder_propagatesStorageError(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	storage := &stubFileStorage{saveErr: errors.New("disk full")}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, &stubDocumentRepo{}, &stubLineItemRepo{}, &stubParsingJobRepo{}, storage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-1", FileName: "order.csv", ContentType: "text/csv", Data: strings.NewReader(""),
	})

	require.Error(t, err)
}
