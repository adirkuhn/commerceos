package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletePickScan_deletesStagedScan(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ID: "scan-1"}}}
	uc := appoms.NewDeletePickScanUseCase(db, sessionRepo, scanRepo)

	err := uc.Execute(context.Background(), "session-1", "scan-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"scan-1"}, scanRepo.deleted)
}

func TestDeletePickScan_rejectsScanFromAnotherSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress}}
	scanRepo := &stubPickScanRepo{listBySession: []omsDomain.PickScan{{ID: "scan-other"}}}
	uc := appoms.NewDeletePickScanUseCase(db, sessionRepo, scanRepo)

	err := uc.Execute(context.Background(), "session-1", "scan-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickScanNotFound)
	assert.Empty(t, scanRepo.deleted)
}

func TestDeletePickScan_rejectsWhenSessionNotInProgress(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusPaused}}
	uc := appoms.NewDeletePickScanUseCase(db, sessionRepo, &stubPickScanRepo{})

	err := uc.Execute(context.Background(), "session-1", "scan-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
}

func TestDeletePickScan_propagatesSessionNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDErr: omsDomain.ErrPickingSessionNotFound}
	uc := appoms.NewDeletePickScanUseCase(db, sessionRepo, &stubPickScanRepo{})

	err := uc.Execute(context.Background(), "missing", "scan-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}
