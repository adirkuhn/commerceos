package iam_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePairingCode_createsPendingCodeWithCallersIdentity(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{}
	uc := appiam.NewGeneratePairingCodeUseCase(db, repo, &eventrecordtest.Outbox{})

	pairing, err := uc.Execute(context.Background(), appiam.GeneratePairingCodeInput{UserID: "user-1", Role: "customer_owner"})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "user-1", pairing.UserID)
	assert.Equal(t, "customer_owner", pairing.Role)
	assert.Equal(t, iamDomain.ScannerPairingCodeStatusPending, pairing.Status)
	assert.Len(t, pairing.Code, 6)
	// Token backs the QR/barcode claim path -- must be high-entropy (128-bit
	// random, base64url has no padding) and distinct from the digit code.
	assert.GreaterOrEqual(t, len(pairing.Token), 20)
	assert.NotEqual(t, pairing.Code, pairing.Token)
	assert.True(t, pairing.ExpiresAt.After(pairing.CreatedAt))
}

func TestGeneratePairingCode_retriesOnCodeCollision(t *testing.T) {
	db := newMockGormDB(t)
	calls := 0
	repo := &stubScannerPairingCodeRepo{}
	// Wrap FindPendingByCode indirectly: first two lookups report a
	// collision, then succeed -- exercised via a small local repo that
	// counts calls, since the real repo takes no per-call override.
	countingRepo := &collidingThenFreeRepo{stubScannerPairingCodeRepo: repo, collideFor: 2, calls: &calls}
	uc := appiam.NewGeneratePairingCodeUseCase(db, countingRepo, &eventrecordtest.Outbox{})

	pairing, err := uc.Execute(context.Background(), appiam.GeneratePairingCodeInput{UserID: "user-1", Role: "customer_owner"})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 3)
	assert.NotEmpty(t, pairing.ID)
}

func TestGeneratePairingCode_propagatesFindError(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findPendingErr: assert.AnError}
	uc := appiam.NewGeneratePairingCodeUseCase(db, repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appiam.GeneratePairingCodeInput{UserID: "user-1", Role: "customer_owner"})

	require.Error(t, err)
}

// collidingThenFreeRepo reports an existing pending code for the first N
// FindPendingByCode calls (forcing GeneratePairingCodeUseCase's retry loop),
// then behaves like the wrapped stub.
type collidingThenFreeRepo struct {
	*stubScannerPairingCodeRepo
	collideFor int
	calls      *int
}

func (r *collidingThenFreeRepo) FindPendingByCode(ctx context.Context, code string) (*iamDomain.ScannerPairingCode, error) {
	*r.calls++
	if *r.calls <= r.collideFor {
		return &iamDomain.ScannerPairingCode{ID: "colliding", Code: code}, nil
	}
	return r.stubScannerPairingCodeRepo.FindPendingByCode(ctx, code)
}
