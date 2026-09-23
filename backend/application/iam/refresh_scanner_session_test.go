package iam_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	appiam "github.com/adirkuhn/commerceos/application/iam"
)

type refreshScannerSessionClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func TestRefreshScannerSessionUseCase_mintsSameClaimsFreshExpiry(t *testing.T) {
	secret := []byte("test-secret")
	uc := appiam.NewRefreshScannerSessionUseCase(secret)

	token, err := uc.Execute("user-1", "tenant-1", "customer_user")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	var claims refreshScannerSessionClaims
	_, err = jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (interface{}, error) {
		return secret, nil
	})
	require.NoError(t, err)

	require.Equal(t, "user-1", claims.UserID)
	require.Equal(t, "tenant-1", claims.TenantID)
	require.Equal(t, "customer_user", claims.Role)
	require.True(t, claims.ExpiresAt.After(time.Now().Add(11*time.Hour)), "expiry should be ~12h out")
}

func TestRefreshScannerSessionUseCase_wrongSecretFailsVerification(t *testing.T) {
	uc := appiam.NewRefreshScannerSessionUseCase([]byte("right-secret"))
	token, err := uc.Execute("user-1", "tenant-1", "customer_user")
	require.NoError(t, err)

	var claims refreshScannerSessionClaims
	_, err = jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	require.Error(t, err)
}
