package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDocumentParsingJob_returnsJob(t *testing.T) {
	db := newMockGormDB(t)
	jobRepo := &stubParsingJobRepo{findResult: &procurementDomain.DocumentParsingJob{
		ID: "job-1", Status: procurementDomain.DocumentParsingJobFailed, ErrorMessage: "AI document parsing is not configured",
	}}
	uc := appprocurement.NewGetDocumentParsingJobUseCase(db, jobRepo)

	job, err := uc.Execute(context.Background(), "job-1")

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.DocumentParsingJobFailed, job.Status)
	assert.Equal(t, "AI document parsing is not configured", job.ErrorMessage)
}

func TestGetDocumentParsingJob_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	jobRepo := &stubParsingJobRepo{findErr: procurementDomain.ErrParsingJobNotFound}
	uc := appprocurement.NewGetDocumentParsingJobUseCase(db, jobRepo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrParsingJobNotFound)
}
