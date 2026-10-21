package shipping_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// shippingProviders is the shippingProviderSupport fake: MyParcel and
// PostNL can ship, nothing else can.
type shippingProviders struct{}

func (shippingProviders) Supports(p integrationsDomain.Provider) bool {
	return p == integrationsDomain.ProviderMyParcel || p == "postnl"
}

type fakeShippingPreferencesRepo struct {
	prefs    *shippingDomain.ShippingPreferences
	upserted *shippingDomain.ShippingPreferences
}

func (f *fakeShippingPreferencesRepo) Get(context.Context) (*shippingDomain.ShippingPreferences, error) {
	return f.prefs, nil
}

func (f *fakeShippingPreferencesRepo) Upsert(_ context.Context, p *shippingDomain.ShippingPreferences) error {
	f.upserted = p
	return nil
}

func adminCtx() context.Context {
	return tenant.WithRole(tenant.WithTenantID(context.Background(), "tenant-1"), tenant.RoleCustomerAdmin)
}

var (
	oldMyParcel = integrationsDomain.Connection{ID: "mp", Provider: integrationsDomain.ProviderMyParcel, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	newPostNL   = integrationsDomain.Connection{ID: "pn", Provider: "postnl", CreatedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}
	bolConn     = integrationsDomain.Connection{ID: "bol", Provider: integrationsDomain.ProviderBol, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
)

func connRepoWith(conns ...integrationsDomain.Connection) *mocks.ConnectionRepo {
	return &mocks.ConnectionRepo{
		ListByTenantFn: func(context.Context) ([]integrationsDomain.Connection, error) { return conns, nil },
		FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) {
			for i := range conns {
				if conns[i].ID == id {
					return &conns[i], nil
				}
			}
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
}

func TestListShippingConnections_onlyShippingProvidersOldestIsDefaultWhenUnset(t *testing.T) {
	uc := appshipping.NewListShippingConnectionsUseCase(newMockGormDB(t, 1), connRepoWith(newPostNL, bolConn, oldMyParcel), &fakeShippingPreferencesRepo{}, shippingProviders{}, withSender())

	got, err := uc.Execute(adminCtx())

	require.NoError(t, err)
	require.Len(t, got.Connections, 2, "the Bol connection can't ship labels")
	assert.Equal(t, "mp", got.Connections[0].ID, "oldest first")
	assert.Equal(t, "pn", got.Connections[1].ID)
	assert.Equal(t, "mp", got.DefaultConnectionID)
	assert.Nil(t, got.ConfiguredConnectionID)
}

func TestListShippingConnections_configuredDefaultWins(t *testing.T) {
	picked := "pn"
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{DefaultConnectionID: &picked}}
	uc := appshipping.NewListShippingConnectionsUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel, newPostNL), prefs, shippingProviders{}, withSender())

	got, err := uc.Execute(adminCtx())

	require.NoError(t, err)
	assert.Equal(t, "pn", got.DefaultConnectionID)
	require.NotNil(t, got.ConfiguredConnectionID)
	assert.Equal(t, "pn", *got.ConfiguredConnectionID)
}

func TestSetDefaultShippingConnection_savesAndRecordsEvent(t *testing.T) {
	prefs := &fakeShippingPreferencesRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewSetDefaultShippingConnectionUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel, newPostNL), prefs, shippingProviders{}, outbox)
	pick := "pn"

	err := uc.Execute(adminCtx(), &pick)

	require.NoError(t, err)
	require.NotNil(t, prefs.upserted)
	assert.Equal(t, "tenant-1", prefs.upserted.TenantID)
	require.NotNil(t, prefs.upserted.DefaultConnectionID)
	assert.Equal(t, "pn", *prefs.upserted.DefaultConnectionID)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "shipping.preferences.updated", e.Subject)
	assert.Equal(t, "tenant-1", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"default_connection_id": {nil, "pn"}}, e.Changes())
}

func TestSetDefaultShippingConnection_nilClearsTheChoice(t *testing.T) {
	was := "pn"
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{DefaultConnectionID: &was}}
	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewSetDefaultShippingConnectionUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel, newPostNL), prefs, shippingProviders{}, outbox)

	err := uc.Execute(adminCtx(), nil)

	require.NoError(t, err)
	require.NotNil(t, prefs.upserted)
	assert.Nil(t, prefs.upserted.DefaultConnectionID)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, map[string][2]any{"default_connection_id": {"pn", nil}}, e.Changes())
}

func TestSetDefaultShippingConnection_rejectsNonShippingConnection(t *testing.T) {
	prefs := &fakeShippingPreferencesRepo{}
	uc := appshipping.NewSetDefaultShippingConnectionUseCase(newMockGormDB(t, 1), connRepoWith(bolConn), prefs, shippingProviders{}, &eventrecordtest.Outbox{})
	pick := "bol"

	err := uc.Execute(adminCtx(), &pick)

	assert.ErrorIs(t, err, shippingDomain.ErrNotAShippingConnection)
	assert.Nil(t, prefs.upserted)
}

func TestSetDefaultShippingConnection_unknownConnectionNotFound(t *testing.T) {
	prefs := &fakeShippingPreferencesRepo{}
	uc := appshipping.NewSetDefaultShippingConnectionUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel), prefs, shippingProviders{}, &eventrecordtest.Outbox{})
	pick := "gone"

	err := uc.Execute(adminCtx(), &pick)

	assert.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound)
	assert.Nil(t, prefs.upserted)
}

func TestSetDefaultShippingConnection_regularUserForbidden(t *testing.T) {
	prefs := &fakeShippingPreferencesRepo{}
	uc := appshipping.NewSetDefaultShippingConnectionUseCase(nil, connRepoWith(oldMyParcel), prefs, shippingProviders{}, &eventrecordtest.Outbox{})
	ctx := tenant.WithRole(tenant.WithTenantID(context.Background(), "tenant-1"), tenant.RoleCustomerUser)
	pick := "mp"

	err := uc.Execute(ctx, &pick)

	assert.ErrorIs(t, err, shippingDomain.ErrShippingPreferencesForbidden)
	assert.Nil(t, prefs.upserted)
}

func TestSetDefaultShippingConnection_keepsTheAddresses(t *testing.T) {
	sender := &shippingDomain.ShippingAddress{Company: "Cantinho", Street: "Damrak", Number: "1", PostalCode: "1012AB", City: "Amsterdam", CC: "NL"}
	prefs := &fakeShippingPreferencesRepo{prefs: &shippingDomain.ShippingPreferences{SenderAddress: sender, ReturnAddress: sender}}
	uc := appshipping.NewSetDefaultShippingConnectionUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel), prefs, shippingProviders{}, &eventrecordtest.Outbox{})
	pick := "mp"

	require.NoError(t, uc.Execute(adminCtx(), &pick))

	require.NotNil(t, prefs.upserted)
	assert.Same(t, sender, prefs.upserted.SenderAddress)
	assert.Same(t, sender, prefs.upserted.ReturnAddress)
}

type fixedAddresses struct{ addrs appshipping.ShipmentAddresses }

func (f fixedAddresses) Resolve(context.Context) (appshipping.ShipmentAddresses, error) {
	return f.addrs, nil
}

func withSender() fixedAddresses {
	return fixedAddresses{appshipping.ShipmentAddresses{Sender: &integrationsDomain.CarrierAddress{Company: "Shop"}}}
}

func TestListShippingConnections_postnlWithoutSenderAddressIsNotReady(t *testing.T) {
	uc := appshipping.NewListShippingConnectionsUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel, newPostNL), &fakeShippingPreferencesRepo{}, shippingProviders{}, fixedAddresses{})

	got, err := uc.Execute(adminCtx())

	require.NoError(t, err)
	require.Len(t, got.Connections, 2)
	assert.Empty(t, got.Connections[0].Missing, "MyParcel prints the sender from its own account")
	assert.Equal(t, []string{appshipping.MissingSenderAddress}, got.Connections[1].Missing)
}

func TestListShippingConnections_withSenderAddressEveryoneIsReady(t *testing.T) {
	uc := appshipping.NewListShippingConnectionsUseCase(newMockGormDB(t, 1), connRepoWith(oldMyParcel, newPostNL), &fakeShippingPreferencesRepo{}, shippingProviders{}, withSender())

	got, err := uc.Execute(adminCtx())

	require.NoError(t, err)
	assert.Empty(t, got.Connections[1].Missing)
}
