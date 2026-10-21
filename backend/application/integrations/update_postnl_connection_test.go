package integrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func postnlConnRepo(t *testing.T, conn integrationsDomain.Connection, updated **integrationsDomain.Connection) *mocks.ConnectionRepo {
	return &mocks.ConnectionRepo{
		FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { c := conn; return &c, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			*updated = c
			return nil
		},
	}
}

func existingPostNL(t *testing.T) integrationsDomain.Connection {
	key, err := crypto.Encrypt(testEncryptionSecret, "old-key")
	require.NoError(t, err)
	return integrationsDomain.Connection{ID: "conn-pn", Provider: integrationsDomain.ProviderPostNL, ConsumerSecretEncrypted: key,
		PostNLCustomerCode: "SLTQ", PostNLCustomerNumber: "10979997", PostNLSandbox: true}
}

func TestUpdatePostNLConnection_switchesToProductionWithNewKey(t *testing.T) {
	var updated *integrationsDomain.Connection
	outbox := &eventrecordtest.Outbox{}
	uc := appintegrations.NewUpdatePostNLConnectionUseCase(newMockGormDB(t), postnlConnRepo(t, existingPostNL(t), &updated), testEncryptionSecret, outbox)

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-pn", appintegrations.UpdatePostNLConnectionInput{
		APIKey: "prod-key", CustomerCode: "sltq", CustomerNumber: "10979997", Sandbox: false,
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.False(t, updated.PostNLSandbox)
	assert.Equal(t, "SLTQ", updated.PostNLCustomerCode)
	key, err := crypto.Decrypt(testEncryptionSecret, updated.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "prod-key", key)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "integrations.connection.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"sandbox": {true, false}, "api_key": {nil, "changed"}}, e.Changes())
}

func TestUpdatePostNLConnection_emptyKeyKeepsTheOldOne(t *testing.T) {
	var updated *integrationsDomain.Connection
	existing := existingPostNL(t)
	uc := appintegrations.NewUpdatePostNLConnectionUseCase(newMockGormDB(t), postnlConnRepo(t, existing, &updated), testEncryptionSecret, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-pn", appintegrations.UpdatePostNLConnectionInput{
		CustomerCode: "SLTQ", CustomerNumber: "99", Sandbox: true,
	})

	require.NoError(t, err)
	assert.Equal(t, existing.ConsumerSecretEncrypted, updated.ConsumerSecretEncrypted)
	assert.Equal(t, "99", updated.PostNLCustomerNumber)
}

func TestUpdatePostNLConnection_rejectsInvalidAccount(t *testing.T) {
	uc := appintegrations.NewUpdatePostNLConnectionUseCase(nil, nil, testEncryptionSecret, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-pn", appintegrations.UpdatePostNLConnectionInput{CustomerCode: "TOOLONG", CustomerNumber: "1"})

	assert.ErrorIs(t, err, integrationsDomain.ErrPostNLAccountInvalid)
}

func TestUpdatePostNLConnection_otherProviderRejected(t *testing.T) {
	var updated *integrationsDomain.Connection
	mp := integrationsDomain.Connection{ID: "conn-mp", Provider: integrationsDomain.ProviderMyParcel}
	uc := appintegrations.NewUpdatePostNLConnectionUseCase(newMockGormDB(t), postnlConnRepo(t, mp, &updated), testEncryptionSecret, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-mp", appintegrations.UpdatePostNLConnectionInput{CustomerCode: "SLTQ", CustomerNumber: "1"})

	assert.ErrorIs(t, err, integrationsDomain.ErrNotPostNLConnection)
	assert.Nil(t, updated)
}

func TestGetPostNLConnection_onlyForPostNL(t *testing.T) {
	var updated *integrationsDomain.Connection
	got, err := appintegrations.NewGetPostNLConnectionUseCase(newMockGormDB(t), postnlConnRepo(t, existingPostNL(t), &updated)).
		Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-pn")
	require.NoError(t, err)
	assert.Equal(t, "SLTQ", got.PostNLCustomerCode)

	_, err = appintegrations.NewGetPostNLConnectionUseCase(newMockGormDB(t), postnlConnRepo(t, integrationsDomain.Connection{Provider: integrationsDomain.ProviderBol}, &updated)).
		Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-bol")
	assert.ErrorIs(t, err, integrationsDomain.ErrNotPostNLConnection)
}
