package procurement_test

import (
	"context"
	"errors"
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
	jobRepo := &stubParsingJobRepo{}
	storage := &stubFileStorage{}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, jobRepo, storage)

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
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, &stubParsingJobRepo{}, &stubFileStorage{})

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
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, &stubParsingJobRepo{}, &stubFileStorage{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-2", FileName: "invoice.pdf", ContentType: "application/pdf", Data: strings.NewReader("%PDF-1.4 fake bytes"),
	})

	require.NoError(t, err)
	assert.True(t, documentRepo.created[0].IsPotentialDuplicate)
	require.Len(t, poRepo.duplicatesSet, 1)
	assert.Equal(t, "po-2", poRepo.duplicatesSet[0].id)
	assert.Equal(t, "po-1", poRepo.duplicatesSet[0].duplicateOfPurchaseOrderID)
}

func TestAttachDocumentToPurchaseOrder_notFlaggedAsDuplicate_whenNoMatchFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	documentRepo := &stubDocumentRepo{}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, &stubParsingJobRepo{}, &stubFileStorage{})

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
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, &stubDocumentRepo{}, &stubParsingJobRepo{}, &stubFileStorage{})

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "missing", FileName: "order.csv", ContentType: "text/csv", Data: strings.NewReader(""),
	})

	assert.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

func TestAttachDocumentToPurchaseOrder_propagatesStorageError(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	storage := &stubFileStorage{saveErr: errors.New("disk full")}
	uc := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, &stubDocumentRepo{}, &stubParsingJobRepo{}, storage)

	_, err := uc.Execute(context.Background(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: "po-1", FileName: "order.csv", ContentType: "text/csv", Data: strings.NewReader(""),
	})

	require.Error(t, err)
}
