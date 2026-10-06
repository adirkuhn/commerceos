package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryDocumentParsingJob_enqueuesANewPendingJobForTheSameDocument(t *testing.T) {
	db := newMockGormDB(t)
	documentRepo := &stubDocumentRepo{findResult: &procurementDomain.PurchaseOrderDocument{
		ID: "document-1", PurchaseOrderID: "po-1",
	}}
	jobRepo := &stubParsingJobRepo{}
	uc := appprocurement.NewRetryDocumentParsingJobUseCase(db, documentRepo, jobRepo, &eventrecordtest.Outbox{})

	job, err := uc.Execute(context.Background(), "document-1")

	require.NoError(t, err)
	require.Len(t, jobRepo.created, 1)
	assert.Equal(t, procurementDomain.DocumentParsingJobPending, job.Status)
	assert.Equal(t, "document-1", job.DocumentID)
	assert.Equal(t, "po-1", job.PurchaseOrderID)
	assert.Equal(t, "job-1", job.ID)
}

func TestRetryDocumentParsingJob_propagatesDocumentNotFound(t *testing.T) {
	db := newMockGormDB(t)
	documentRepo := &stubDocumentRepo{findErr: procurementDomain.ErrDocumentNotFound}
	jobRepo := &stubParsingJobRepo{}
	uc := appprocurement.NewRetryDocumentParsingJobUseCase(db, documentRepo, jobRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrDocumentNotFound)
	assert.Empty(t, jobRepo.created)
}
