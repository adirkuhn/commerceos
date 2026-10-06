package procurement_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestDeletePurchaseOrderDocument_deletesRowAndStoredFile(t *testing.T) {
	documentRepo := &stubDocumentRepo{findResult: &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://uploads.example.com/tenant-1/abc.pdf"}}
	storage := &stubFileStorage{}
	uc := appprocurement.NewDeletePurchaseOrderDocumentUseCase(newMockGormDB(t), documentRepo, storage, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "document-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"document-1"}, documentRepo.deleted)
	assert.Equal(t, "https://uploads.example.com/tenant-1/abc.pdf", storage.deletedURL)
}

func TestDeletePurchaseOrderDocument_propagatesNotFound(t *testing.T) {
	documentRepo := &stubDocumentRepo{findErr: procurementDomain.ErrDocumentNotFound}
	uc := appprocurement.NewDeletePurchaseOrderDocumentUseCase(newMockGormDB(t), documentRepo, &stubFileStorage{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrDocumentNotFound)
}

func TestDeletePurchaseOrderDocument_doesNotDeleteStoredFile_whenRowDeleteFails(t *testing.T) {
	documentRepo := &stubDocumentRepo{
		findResult: &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://uploads.example.com/tenant-1/abc.pdf"},
		deleteErr:  errors.New("db down"),
	}
	storage := &stubFileStorage{}
	uc := appprocurement.NewDeletePurchaseOrderDocumentUseCase(newMockGormDB(t), documentRepo, storage, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "document-1")

	require.Error(t, err)
	assert.Empty(t, storage.deletedURL, "must never delete the file when the row delete itself failed")
}
