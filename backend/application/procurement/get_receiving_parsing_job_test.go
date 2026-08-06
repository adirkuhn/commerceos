package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReceivingParsingJob_returnsJob(t *testing.T) {
	db := newMockGormDB(t)
	jobRepo := &stubReceivingParsingJobRepo{findResult: &procurementDomain.ReceivingParsingJob{
		ID: "receiving-job-1", Status: procurementDomain.ReceivingParsingJobFailed, ErrorMessage: "AI document parsing is not configured",
	}}
	uc := appprocurement.NewGetReceivingParsingJobUseCase(db, jobRepo)

	job, err := uc.Execute(context.Background(), "receiving-job-1")

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.ReceivingParsingJobFailed, job.Status)
	assert.Equal(t, "AI document parsing is not configured", job.ErrorMessage)
}

func TestGetReceivingParsingJob_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	jobRepo := &stubReceivingParsingJobRepo{findErr: procurementDomain.ErrParsingJobNotFound}
	uc := appprocurement.NewGetReceivingParsingJobUseCase(db, jobRepo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrParsingJobNotFound)
}
