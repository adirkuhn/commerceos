package integrations_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/internal/state"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

var testStateSecret = []byte("test-secret")

func TestStartWooConnect_buildsAuthorizeURLWithState(t *testing.T) {
	uc := appintegrations.NewStartWooConnectUseCase(testStateSecret, "https://api.commerceos.dev.localhost", "https://commerceos.dev.localhost")
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")

	result, err := uc.Execute(ctx, appintegrations.StartWooConnectInput{StoreURL: "https://mystore.com"})
	require.NoError(t, err)
	require.NotEmpty(t, result.AuthorizeURL)

	parsed, err := url.Parse(result.AuthorizeURL)
	require.NoError(t, err)
	assert.Equal(t, "mystore.com", parsed.Host)
	assert.Equal(t, "/wc-auth/v1/authorize", parsed.Path)

	q := parsed.Query()
	assert.Equal(t, "https://api.commerceos.dev.localhost/api/v1/integrations/woocommerce/connect/callback", q.Get("callback_url"))
	assert.Equal(t, "https://commerceos.dev.localhost/integrations", q.Get("return_url"))
	require.NotEmpty(t, q.Get("user_id"))

	got, err := state.ParseConnectState(testStateSecret, q.Get("user_id"), time.Now())
	require.NoError(t, err)
	assert.Equal(t, "tenant-1", got.TenantID)
	assert.Equal(t, "https://mystore.com", got.StoreURL)
}

func TestStartWooConnect_rejectsInvalidStoreURL(t *testing.T) {
	uc := appintegrations.NewStartWooConnectUseCase(testStateSecret, "https://api.commerceos.dev.localhost", "https://commerceos.dev.localhost")
	ctx := tenant.WithTenantID(context.Background(), "tenant-1")

	_, err := uc.Execute(ctx, appintegrations.StartWooConnectInput{StoreURL: "not-a-url"})
	assert.Error(t, err)
}

func TestStartWooConnect_rejectsMissingTenant(t *testing.T) {
	uc := appintegrations.NewStartWooConnectUseCase(testStateSecret, "https://api.commerceos.dev.localhost", "https://commerceos.dev.localhost")

	_, err := uc.Execute(context.Background(), appintegrations.StartWooConnectInput{StoreURL: "https://mystore.com"})
	assert.Error(t, err)
}

func TestStartWooConnect_superAdminWithTenantContextSucceeds(t *testing.T) {
	// middleware.JWT sets both TenantID and the super-admin flag from lm2's
	// claims (lm2 always resolves a company on login, even for super_admin)
	// — this use case only needs the tenant, regardless of the flag.
	uc := appintegrations.NewStartWooConnectUseCase(testStateSecret, "https://api.commerceos.dev.localhost", "https://commerceos.dev.localhost")
	ctx := tenant.WithSuperAdmin(tenant.WithTenantID(context.Background(), "tenant-99"))

	result, err := uc.Execute(ctx, appintegrations.StartWooConnectInput{StoreURL: "https://mystore.com"})
	require.NoError(t, err)

	parsed, err := url.Parse(result.AuthorizeURL)
	require.NoError(t, err)
	got, err := state.ParseConnectState(testStateSecret, parsed.Query().Get("user_id"), time.Now())
	require.NoError(t, err)
	assert.Equal(t, "tenant-99", got.TenantID)
}
