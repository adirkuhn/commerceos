package moneybird_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func TestConnect_checksTheTokenThenStoresItEncrypted(t *testing.T) {
	conns := &fakeConnections{}
	settings := newFakeSettings()
	factory := &fakeFactory{client: &fakeClient{administrations: []mb.Administration{{ID: "adm-1", Name: "Cantinho"}}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appmb.NewConnectUseCase(mockDB(t, 1), conns, settings, factory, outbox, testSecret)

	conn, err := uc.Execute(tenantCtx(), appmb.ConnectInput{APIToken: "  tok-1  "})

	require.NoError(t, err)
	assert.Equal(t, []string{"tok-1"}, factory.tokens)
	require.Len(t, conns.created, 1)
	assert.Equal(t, integrationsDomain.ProviderMoneybird, conns.created[0].Provider)
	plain, err := crypto.Decrypt(testSecret, conns.created[0].ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "tok-1", plain)
	assert.Equal(t, "conn-new", conn.ID)
	// The only administration is picked straight away.
	assert.Equal(t, "adm-1", settings.byConn["conn-new"].AdministrationID)
	assert.Equal(t, "Cantinho", settings.byConn["conn-new"].AdministrationName)
	assert.False(t, settings.byConn["conn-new"].AutoSync, "auto sync starts off")
	assert.Equal(t, []string{"integrations.connection.connected"}, outbox.Subjects())
}

// COM-15: reconnecting after a disconnect takes the old row back, so the
// administration, mappings and auto-sync choice survive.
func TestConnect_afterADisconnectReusesTheConnectionAndKeepsItsSettings(t *testing.T) {
	conns := &fakeConnections{disconnected: &integrationsDomain.Connection{ID: "conn-old", Provider: integrationsDomain.ProviderMoneybird,
		Status: integrationsDomain.StatusDisconnected}}
	settings := newFakeSettings(mb.Settings{ConnectionID: "conn-old", AdministrationID: "adm-1", AutoSync: true})
	factory := &fakeFactory{client: &fakeClient{administrations: []mb.Administration{{ID: "adm-2", Name: "Other"}}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appmb.NewConnectUseCase(mockDB(t, 1), conns, settings, factory, outbox, testSecret)

	conn, err := uc.Execute(tenantCtx(), appmb.ConnectInput{APIToken: "tok-2"})

	require.NoError(t, err)
	assert.Equal(t, "conn-old", conn.ID)
	assert.Empty(t, conns.created)
	require.Len(t, conns.updated, 1)
	assert.Equal(t, integrationsDomain.StatusConnected, conns.updated[0].Status)
	assert.Equal(t, "adm-1", settings.byConn["conn-old"].AdministrationID, "settings kept")
	assert.True(t, settings.byConn["conn-old"].AutoSync)
	assert.Equal(t, []string{"integrations.connection.connected"}, outbox.Subjects())
}

func TestConnect_severalAdministrationsLeavesTheChoiceToTheUser(t *testing.T) {
	settings := newFakeSettings()
	factory := &fakeFactory{client: &fakeClient{administrations: []mb.Administration{{ID: "a"}, {ID: "b"}}}}
	uc := appmb.NewConnectUseCase(mockDB(t, 1), &fakeConnections{}, settings, factory, &eventrecordtest.Outbox{}, testSecret)

	_, err := uc.Execute(tenantCtx(), appmb.ConnectInput{APIToken: "tok"})

	require.NoError(t, err)
	assert.Empty(t, settings.byConn["conn-new"].AdministrationID)
}

func TestConnect_rejectedTokenStoresNothing(t *testing.T) {
	conns := &fakeConnections{}
	factory := &fakeFactory{client: &fakeClient{err: mb.ErrTokenInvalid}}
	uc := appmb.NewConnectUseCase(mockDB(t, 1), conns, newFakeSettings(), factory, &eventrecordtest.Outbox{}, testSecret)

	_, err := uc.Execute(tenantCtx(), appmb.ConnectInput{APIToken: "bad"})

	assert.True(t, errors.Is(err, mb.ErrTokenInvalid))
	assert.Empty(t, conns.created)
}

func TestConnect_blankToken(t *testing.T) {
	uc := appmb.NewConnectUseCase(mockDB(t, 1), &fakeConnections{}, newFakeSettings(), &fakeFactory{client: &fakeClient{}}, &eventrecordtest.Outbox{}, testSecret)
	_, err := uc.Execute(tenantCtx(), appmb.ConnectInput{APIToken: "  "})
	assert.True(t, errors.Is(err, mb.ErrTokenInvalid))
}

// One Moneybird connection per shop: connecting again swaps the token.
func TestConnect_againReplacesTheToken(t *testing.T) {
	conns := connected(t, "old")
	uc := appmb.NewConnectUseCase(mockDB(t, 1), conns, newFakeSettings(configuredSettings()),
		&fakeFactory{client: &fakeClient{administrations: []mb.Administration{{ID: "adm-1"}}}}, &eventrecordtest.Outbox{}, testSecret)

	conn, err := uc.Execute(tenantCtx(), appmb.ConnectInput{APIToken: "new"})

	require.NoError(t, err)
	assert.Equal(t, "conn-mb", conn.ID)
	assert.Empty(t, conns.created)
	require.Len(t, conns.updated, 1)
	plain, _ := crypto.Decrypt(testSecret, conns.updated[0].ConsumerSecretEncrypted)
	assert.Equal(t, "new", plain)
}

func TestSetup_listsWhatTheSettingsChooseFrom(t *testing.T) {
	client := &fakeClient{
		administrations: []mb.Administration{{ID: "adm-1", Name: "Cantinho"}},
		taxRates:        []mb.TaxRate{{ID: "tr-21", Name: "21%", Percentage: "21.0"}},
		ledgers:         []mb.LedgerAccount{{ID: "led-1", Name: "Omzet"}},
	}
	factory := &fakeFactory{client: client}
	uc := appmb.NewGetSetupUseCase(mockDB(t, 1), connected(t, "tok"), newFakeSettings(configuredSettings()), factory, testSecret)

	got, err := uc.Execute(tenantCtx())

	require.NoError(t, err)
	assert.Equal(t, "conn-mb", got.ConnectionID)
	assert.Equal(t, "adm-1", got.Settings.AdministrationID)
	assert.Len(t, got.Administrations, 1)
	assert.Len(t, got.TaxRates, 1)
	assert.Len(t, got.LedgerAccounts, 1)
	assert.Contains(t, factory.adminIDs, "adm-1")
	assert.Equal(t, "tok", factory.tokens[0])
}

func TestSetup_withoutAdministrationOnlyListsAdministrations(t *testing.T) {
	s := configuredSettings()
	s.AdministrationID = ""
	uc := appmb.NewGetSetupUseCase(mockDB(t, 1), connected(t, "tok"), newFakeSettings(s),
		&fakeFactory{client: &fakeClient{administrations: []mb.Administration{{ID: "a"}}, taxRates: []mb.TaxRate{{ID: "x"}}}}, testSecret)

	got, err := uc.Execute(tenantCtx())

	require.NoError(t, err)
	assert.Len(t, got.Administrations, 1)
	assert.Empty(t, got.TaxRates)
}

func TestSetup_notConnected(t *testing.T) {
	uc := appmb.NewGetSetupUseCase(mockDB(t, 1), &fakeConnections{}, newFakeSettings(), &fakeFactory{client: &fakeClient{}}, testSecret)
	_, err := uc.Execute(tenantCtx())
	assert.True(t, errors.Is(err, mb.ErrNotConnected))
}

func ptr[T any](v T) *T { return &v }

func newUpdateUC(t *testing.T, settings *fakeSettings, client *fakeClient, outbox *eventrecordtest.Outbox) *appmb.UpdateSettingsUseCase {
	return appmb.NewUpdateSettingsUseCase(mockDB(t, 2), connected(t, "tok"), settings, &fakeFactory{client: client}, outbox, testSecret)
}

func mbLists() *fakeClient {
	return &fakeClient{
		administrations: []mb.Administration{{ID: "adm-1", Name: "Cantinho"}, {ID: "adm-2", Name: "Other"}},
		taxRates:        []mb.TaxRate{{ID: "tr-21"}, {ID: "tr-9"}, {ID: "tr-0"}},
		ledgers:         []mb.LedgerAccount{{ID: "led-1"}, {ID: "led-2"}},
	}
}

// Sections send only their own fields: turning auto sync on keeps the mappings.
func TestUpdateSettings_onlyTouchesTheFieldsSent(t *testing.T) {
	settings := newFakeSettings(configuredSettings())
	outbox := &eventrecordtest.Outbox{}

	got, err := newUpdateUC(t, settings, mbLists(), outbox).Execute(tenantCtx(), appmb.UpdateSettingsInput{AutoSync: ptr(true)})

	require.NoError(t, err)
	assert.True(t, got.AutoSync)
	assert.Equal(t, configuredSettings().TaxRates, got.TaxRates)
	assert.Equal(t, "led-1", got.DefaultLedgerAccountID)
	assert.Equal(t, []string{"integrations.moneybird_settings.updated"}, outbox.Subjects())
}

func TestUpdateSettings_mappingsMustExistInMoneybird(t *testing.T) {
	settings := newFakeSettings(configuredSettings())
	uc := newUpdateUC(t, settings, mbLists(), &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenantCtx(), appmb.UpdateSettingsInput{TaxRates: ptr([]mb.TaxRateMapping{{RateBasisPoints: 2100, MoneybirdTaxRateID: "nope"}})})
	assert.True(t, errors.Is(err, appmb.ErrUnknownMoneybirdID))

	_, err = uc.Execute(tenantCtx(), appmb.UpdateSettingsInput{DefaultLedgerAccountID: ptr("nope")})
	assert.True(t, errors.Is(err, appmb.ErrUnknownMoneybirdID))
	assert.Empty(t, settings.upserts)
}

func TestUpdateSettings_oneMappingPerRateAndMethod(t *testing.T) {
	uc := newUpdateUC(t, newFakeSettings(configuredSettings()), mbLists(), &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenantCtx(), appmb.UpdateSettingsInput{TaxRates: ptr([]mb.TaxRateMapping{
		{RateBasisPoints: 900, MoneybirdTaxRateID: "tr-9"}, {RateBasisPoints: 900, MoneybirdTaxRateID: "tr-21"},
	})})
	assert.True(t, errors.Is(err, appmb.ErrDuplicateMapping))

	_, err = uc.Execute(tenantCtx(), appmb.UpdateSettingsInput{PaymentMethods: ptr([]mb.PaymentMethodMapping{
		{Method: "bacs", MoneybirdPaymentMethod: "bank_transfer"}, {Method: "bacs", MoneybirdPaymentMethod: "pin"},
	})})
	assert.True(t, errors.Is(err, appmb.ErrDuplicateMapping))
}

func TestUpdateSettings_paymentMethodMustBeAMoneybirdMethod(t *testing.T) {
	uc := newUpdateUC(t, newFakeSettings(configuredSettings()), mbLists(), &eventrecordtest.Outbox{})
	_, err := uc.Execute(tenantCtx(), appmb.UpdateSettingsInput{PaymentMethods: ptr([]mb.PaymentMethodMapping{{Method: "bacs", MoneybirdPaymentMethod: "bitcoin"}})})
	assert.True(t, errors.Is(err, appmb.ErrUnknownMoneybirdID))
}

// Ledger accounts, tax rates and contacts belong to one administration.
func TestUpdateSettings_switchingAdministrationClearsItsIDs(t *testing.T) {
	s := configuredSettings()
	s.GuestContactID = "guest"
	settings := newFakeSettings(s)

	got, err := newUpdateUC(t, settings, mbLists(), &eventrecordtest.Outbox{}).Execute(tenantCtx(), appmb.UpdateSettingsInput{AdministrationID: ptr("adm-2")})

	require.NoError(t, err)
	assert.Equal(t, "adm-2", got.AdministrationID)
	assert.Equal(t, "Other", got.AdministrationName)
	assert.Empty(t, got.DefaultLedgerAccountID)
	assert.Empty(t, got.TaxRates)
	assert.Empty(t, got.GuestContactID)
	assert.Equal(t, s.PaymentMethods, got.PaymentMethods, "payment methods are Moneybird's own names, not per administration")
}

func TestUpdateSettings_unknownAdministration(t *testing.T) {
	_, err := newUpdateUC(t, newFakeSettings(configuredSettings()), mbLists(), &eventrecordtest.Outbox{}).
		Execute(tenantCtx(), appmb.UpdateSettingsInput{AdministrationID: ptr("adm-9")})
	assert.True(t, errors.Is(err, appmb.ErrUnknownMoneybirdID))
}

// "" maps orders without a payment method -- allowed once, like any method.
func TestUpdateSettings_ordersWithoutAPaymentMethodCanBeMapped(t *testing.T) {
	settings := newFakeSettings(configuredSettings())
	got, err := newUpdateUC(t, settings, mbLists(), &eventrecordtest.Outbox{}).Execute(tenantCtx(), appmb.UpdateSettingsInput{
		PaymentMethods: ptr([]mb.PaymentMethodMapping{{Method: "", MoneybirdPaymentMethod: "bank_transfer"}}),
	})
	require.NoError(t, err)
	assert.Equal(t, []mb.PaymentMethodMapping{{Method: "", MoneybirdPaymentMethod: "bank_transfer"}}, got.PaymentMethods)
}
