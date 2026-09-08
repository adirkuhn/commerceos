package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
)

func TestLogout_deletesTheSession(t *testing.T) {
	sessionRepo := &fakeCustomerSessionRepo{}
	uc := appstorefront.NewLogoutUseCase(newMockGormDB(t, 1), sessionRepo)

	err := uc.Execute(context.Background(), "SESSION123")

	require.NoError(t, err)
	require.Equal(t, "SESSION123", sessionRepo.deletedToken)
}
