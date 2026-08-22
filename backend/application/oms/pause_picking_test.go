package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPausePicking_pausesInProgressSession(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusInProgress}}
	uc := appoms.NewPausePickingUseCase(db, sessionRepo)

	session, err := uc.Execute(context.Background(), "session-1")

	require.NoError(t, err)
	require.Len(t, sessionRepo.updated, 1)
	assert.Equal(t, omsDomain.PickingSessionStatusPaused, sessionRepo.updated[0].Status)
	assert.NotNil(t, sessionRepo.updated[0].PausedAt)
	assert.Equal(t, omsDomain.PickingSessionStatusPaused, session.Status)
}

func TestPausePicking_rejectsWhenNotInProgress(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "session-1", Status: omsDomain.PickingSessionStatusCompleted}}
	uc := appoms.NewPausePickingUseCase(db, sessionRepo)

	_, err := uc.Execute(context.Background(), "session-1")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotActive)
	assert.Empty(t, sessionRepo.updated)
}

func TestPausePicking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	sessionRepo := &stubPickingSessionRepo{findByIDErr: omsDomain.ErrPickingSessionNotFound}
	uc := appoms.NewPausePickingUseCase(db, sessionRepo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrPickingSessionNotFound)
}
