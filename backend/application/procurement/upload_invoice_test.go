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

func TestUploadInvoice_savesRecordAndEnqueuesJob(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	recRepo := &stubReceivingRecordRepo{}
	jobRepo := &stubReceivingParsingJobRepo{}
	storage := &stubFileStorage{}
	uc := appprocurement.NewUploadInvoiceUseCase(db, poRepo, recRepo, jobRepo, storage)

	job, err := uc.Execute(context.Background(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: "po-1", FileName: "invoice.csv", ContentType: "text/csv", Data: strings.NewReader("sku,qty\nA,1"),
	})

	require.NoError(t, err)
	require.Len(t, recRepo.created, 1)
	assert.Equal(t, "po-1", recRepo.created[0].PurchaseOrderID)
	assert.Equal(t, "invoice.csv", recRepo.created[0].InvoiceFileName)
	assert.Equal(t, procurementDomain.ReceivingRecordStatusPendingReview, recRepo.created[0].Status)
	require.Len(t, jobRepo.created, 1)
	assert.Equal(t, "receiving-record-1", jobRepo.created[0].ReceivingRecordID)
	assert.Equal(t, procurementDomain.ReceivingParsingJobPending, job.Status)
	assert.Equal(t, "sku,qty\nA,1", string(storage.savedData))
}

func TestUploadInvoice_flagsPotentialDuplicate_whenMatchFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1")}}
	recRepo := &stubReceivingRecordRepo{duplicateCandidates: []procurementDomain.ReceivingRecord{{ID: "receiving-record-0"}}}
	jobRepo := &stubReceivingParsingJobRepo{}
	storage := &stubFileStorage{}
	uc := appprocurement.NewUploadInvoiceUseCase(db, poRepo, recRepo, jobRepo, storage)

	_, err := uc.Execute(context.Background(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: "po-1", FileName: "invoice.csv", ContentType: "text/csv", Data: strings.NewReader("sku,qty\nA,1"),
	})

	require.NoError(t, err)
	require.Len(t, recRepo.created, 1)
	assert.True(t, recRepo.created[0].IsPotentialDuplicate)
	assert.NotEmpty(t, recRepo.created[0].ContentHash)
	assert.Equal(t, "po-1", recRepo.gotDuplicatePOID)
	require.NotNil(t, recRepo.gotDuplicateSupplierID)
	assert.Equal(t, "supplier-1", *recRepo.gotDuplicateSupplierID)
	// The uploaded bytes still reach storage untouched -- buffering to hash
	// them must not corrupt or truncate what gets saved.
	assert.Equal(t, "sku,qty\nA,1", string(storage.savedData))
}

func TestUploadInvoice_notFlaggedAsDuplicate_whenNoMatchFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	recRepo := &stubReceivingRecordRepo{}
	uc := appprocurement.NewUploadInvoiceUseCase(db, poRepo, recRepo, &stubReceivingParsingJobRepo{}, &stubFileStorage{})

	_, err := uc.Execute(context.Background(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: "po-1", FileName: "invoice.csv", ContentType: "text/csv", Data: strings.NewReader("sku,qty\nA,1"),
	})

	require.NoError(t, err)
	require.Len(t, recRepo.created, 1)
	assert.False(t, recRepo.created[0].IsPotentialDuplicate)
}

func TestUploadInvoice_propagatesPurchaseOrderNotFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewUploadInvoiceUseCase(db, poRepo, &stubReceivingRecordRepo{}, &stubReceivingParsingJobRepo{}, &stubFileStorage{})

	_, err := uc.Execute(context.Background(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: "missing", FileName: "invoice.csv", ContentType: "text/csv", Data: strings.NewReader(""),
	})

	assert.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

func TestUploadInvoice_propagatesStorageError(t *testing.T) {
	db := newMockGormDBN(t, 2)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	storage := &stubFileStorage{saveErr: errors.New("disk full")}
	uc := appprocurement.NewUploadInvoiceUseCase(db, poRepo, &stubReceivingRecordRepo{}, &stubReceivingParsingJobRepo{}, storage)

	_, err := uc.Execute(context.Background(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: "po-1", FileName: "invoice.csv", ContentType: "text/csv", Data: strings.NewReader(""),
	})

	require.Error(t, err)
}
