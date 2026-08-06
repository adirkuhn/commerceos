package iam_test

import (
	"context"
	"testing"

	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPairingCodeStatus_returnsRepoResult(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findByIDResult: &iamDomain.ScannerPairingCode{ID: "pairing-1", Status: iamDomain.ScannerPairingCodeStatusClaimed}}
	uc := appiam.NewGetPairingCodeStatusUseCase(db, repo)

	pairing, err := uc.Execute(context.Background(), "pairing-1")

	require.NoError(t, err)
	assert.Equal(t, iamDomain.ScannerPairingCodeStatusClaimed, pairing.Status)
}

func TestGetPairingCodeStatus_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findByIDErr: iamDomain.ErrPairingCodeNotFound}
	uc := appiam.NewGetPairingCodeStatusUseCase(db, repo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, iamDomain.ErrPairingCodeNotFound)
}
