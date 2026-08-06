package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func withTenantCtx(tenantID string) context.Context {
	return tenant.WithTenantID(context.Background(), tenantID)
}

func TestConnectAIProvider_encryptsKeyAndUpserts(t *testing.T) {
	var upserted *integrationsDomain.AIConnection
	repo := &mocks.AIConnectionRepo{
		UpsertFn: func(_ context.Context, c *integrationsDomain.AIConnection) error {
			upserted = c
			return nil
		},
	}
	uc := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), repo, "test-secret")

	conn, err := uc.Execute(withTenantCtx("tenant-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProviderOpenAI, APIKey: "sk-test-key", Model: "gpt-4o",
	})

	require.NoError(t, err)
	require.NotNil(t, upserted)
	assert.Equal(t, "tenant-1", conn.TenantID)
	assert.Equal(t, integrationsDomain.AIProviderOpenAI, conn.Provider)
	assert.NotEmpty(t, conn.APIKeyEncrypted)
	assert.NotEqual(t, "sk-test-key", conn.APIKeyEncrypted, "the raw key must never be stored/returned as-is")
	assert.Equal(t, integrationsDomain.StatusConnected, conn.Status)
}

func TestConnectAIProvider_ollamaAllowsEmptyAPIKey(t *testing.T) {
	repo := &mocks.AIConnectionRepo{
		UpsertFn: func(_ context.Context, _ *integrationsDomain.AIConnection) error { return nil },
	}
	uc := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), repo, "test-secret")

	conn, err := uc.Execute(withTenantCtx("tenant-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProviderOllama, BaseURL: "http://localhost:11434",
	})

	require.NoError(t, err)
	assert.Empty(t, conn.APIKeyEncrypted)
}

func TestConnectAIProvider_ollamaRequiresBaseURL(t *testing.T) {
	uc := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), &mocks.AIConnectionRepo{}, "test-secret")

	_, err := uc.Execute(withTenantCtx("tenant-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProviderOllama,
	})

	assert.ErrorIs(t, err, integrationsDomain.ErrBaseURLRequired)
}

func TestConnectAIProvider_hostedProviderRequiresAPIKey(t *testing.T) {
	uc := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), &mocks.AIConnectionRepo{}, "test-secret")

	_, err := uc.Execute(withTenantCtx("tenant-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProviderAnthropic,
	})

	assert.ErrorIs(t, err, integrationsDomain.ErrAPIKeyRequired)
}

func TestConnectAIProvider_rejectsUnsupportedProvider(t *testing.T) {
	uc := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), &mocks.AIConnectionRepo{}, "test-secret")

	_, err := uc.Execute(withTenantCtx("tenant-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProvider("deepseek"), APIKey: "key",
	})

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}

func TestConnectAIProvider_propagatesUpsertError(t *testing.T) {
	repo := &mocks.AIConnectionRepo{
		UpsertFn: func(_ context.Context, _ *integrationsDomain.AIConnection) error { return errors.New("db down") },
	}
	uc := appintegrations.NewConnectAIProviderUseCase(newMockGormDB(t), repo, "test-secret")

	_, err := uc.Execute(withTenantCtx("tenant-1"), appintegrations.ConnectAIProviderInput{
		Provider: integrationsDomain.AIProviderOpenAI, APIKey: "sk-test",
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}
