package iam_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
)

type parsedClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func TestClaimPairingCode_mintsTokenWithPairingIdentityAndMarksClaimed(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findPendingResult: &iamDomain.ScannerPairingCode{
		ID: "pairing-1", TenantID: "tenant-1", UserID: "user-1", Role: "customer_owner",
		Status: iamDomain.ScannerPairingCodeStatusPending,
	}}
	secret := []byte("test-secret")
	uc := appiam.NewClaimPairingCodeUseCase(db, repo, secret, &eventrecordtest.Outbox{})

	tokenStr, err := uc.Execute(context.Background(), "123456")

	require.NoError(t, err)
	assert.Equal(t, []string{"pairing-1"}, repo.markClaimedCalls)

	claims := &parsedClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(*jwt.Token) (any, error) { return secret, nil })
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, "customer_owner", claims.Role)
}

func TestClaimPairingCode_rejectsInvalidOrExpiredCode(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findPendingResult: nil}
	uc := appiam.NewClaimPairingCodeUseCase(db, repo, []byte("test-secret"), &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "000000")

	assert.ErrorIs(t, err, iamDomain.ErrPairingCodeInvalid)
	assert.Empty(t, repo.markClaimedCalls)
}

func TestClaimPairingCodeByToken_mintsTokenWithPairingIdentityAndMarksClaimed(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findPendingByTokenResult: &iamDomain.ScannerPairingCode{
		ID: "pairing-1", TenantID: "tenant-1", UserID: "user-1", Role: "customer_owner",
		Status: iamDomain.ScannerPairingCodeStatusPending,
	}}
	secret := []byte("test-secret")
	uc := appiam.NewClaimPairingCodeUseCase(db, repo, secret, &eventrecordtest.Outbox{})

	tokenStr, err := uc.ExecuteByToken(context.Background(), "opaque-abc")

	require.NoError(t, err)
	assert.Equal(t, []string{"pairing-1"}, repo.markClaimedCalls)

	claims := &parsedClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(*jwt.Token) (any, error) { return secret, nil })
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, "customer_owner", claims.Role)
}

func TestClaimPairingCodeByToken_rejectsInvalidOrExpiredToken(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{findPendingByTokenResult: nil}
	uc := appiam.NewClaimPairingCodeUseCase(db, repo, []byte("test-secret"), &eventrecordtest.Outbox{})

	_, err := uc.ExecuteByToken(context.Background(), "does-not-exist")

	assert.ErrorIs(t, err, iamDomain.ErrPairingCodeInvalid)
	assert.Empty(t, repo.markClaimedCalls)
}

func TestClaimPairingCode_propagatesMarkClaimedError(t *testing.T) {
	db := newMockGormDB(t)
	repo := &stubScannerPairingCodeRepo{
		findPendingResult: &iamDomain.ScannerPairingCode{ID: "pairing-1", TenantID: "tenant-1", UserID: "user-1", Role: "customer_owner"},
		markClaimedErr:    assert.AnError,
	}
	uc := appiam.NewClaimPairingCodeUseCase(db, repo, []byte("test-secret"), &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "123456")

	require.Error(t, err)
}
