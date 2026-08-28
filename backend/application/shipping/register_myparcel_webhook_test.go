package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestRegisterMyParcelWebhook_buildsCallbackURLWithTenantAndToken(t *testing.T) {
	conn := &integrationsDomain.Connection{
		ID: "conn-1", TenantID: "tenant-1",
		ConsumerSecretEncrypted: encryptedFor(t, "api-key"),
		WebhookSecretEncrypted:  encryptedFor(t, "webhook-secret-xyz"),
	}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	var gotURL string
	client := &fakeMyParcelClient{registerWebhookFn: func(_ context.Context, callbackURL string) error {
		gotURL = callbackURL
		return nil
	}}
	factory := &fakeMyParcelClientFactory{client: client}

	uc := appshipping.NewRegisterMyParcelWebhookUseCase(newMockGormDB(t, 1), connRepo, factory, testEncSecret, "https://api.example.com")

	err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/api/v1/webhooks/v1/tenant-1/myparcel/webhook-secret-xyz", gotURL)
}

func TestRegisterMyParcelWebhook_noSecretSet_returnsErrWebhookSecretNotSet(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", TenantID: "tenant-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	uc := appshipping.NewRegisterMyParcelWebhookUseCase(newMockGormDB(t, 1), connRepo, &fakeMyParcelClientFactory{}, testEncSecret, "https://api.example.com")

	err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-1")

	require.ErrorIs(t, err, shippingDomain.ErrWebhookSecretNotSet)
}
