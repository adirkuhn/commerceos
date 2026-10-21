package integrations_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type recordingOverrides struct {
	upserted *shippingDomain.ShippingCarrierOverride
}

func (r *recordingOverrides) Upsert(_ context.Context, o *shippingDomain.ShippingCarrierOverride) error {
	r.upserted = o
	return nil
}

type recordingSchedules struct{ upserted *channelsDomain.SyncSchedule }

func (r *recordingSchedules) Upsert(_ context.Context, s *channelsDomain.SyncSchedule) error {
	r.upserted = s
	return nil
}

type postnlConnectFixture struct {
	created   *integrationsDomain.Connection
	settings  *shippingDomain.ShippingConnectionSettings
	overrides recordingOverrides
	schedules recordingSchedules
	subject   string
}

func (f *postnlConnectFixture) useCase(t *testing.T) *appintegrations.ConnectPostNLUseCase {
	connRepo := &mocks.ConnectionRepo{CreateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
		c.ID = "conn-postnl-1"
		f.created = c
		return nil
	}}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{UpsertFn: func(_ context.Context, s *shippingDomain.ShippingConnectionSettings) error {
		f.settings = s
		return nil
	}}
	outboxRepo := &mocks.OutboxRepo{EnqueueFn: func(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
		f.subject = subject
		return nil
	}}
	return appintegrations.NewConnectPostNLUseCase(newMockGormDB(t), connRepo, settingsRepo, &f.overrides, &f.schedules, outboxRepo, testEncryptionSecret)
}

func validPostNLInput() appintegrations.ConnectPostNLInput {
	return appintegrations.ConnectPostNLInput{APIKey: "key-1", CustomerCode: " devc ", CustomerNumber: "11223344", Sandbox: true}
}

func TestConnectPostNL_createsConnectionWithAccountAndEncryptedKey(t *testing.T) {
	f := &postnlConnectFixture{}

	conn, err := f.useCase(t).Execute(tenant.WithTenantID(context.Background(), "tenant-1"), validPostNLInput())

	require.NoError(t, err)
	assert.Equal(t, "conn-postnl-1", conn.ID)
	require.NotNil(t, f.created)
	assert.Equal(t, integrationsDomain.ProviderPostNL, f.created.Provider)
	assert.Equal(t, integrationsDomain.StatusConnected, f.created.Status)
	assert.Equal(t, "DEVC", f.created.PostNLCustomerCode, "trimmed and upper-cased")
	assert.Equal(t, "11223344", f.created.PostNLCustomerNumber)
	assert.True(t, f.created.PostNLSandbox)
	key, err := crypto.Decrypt(testEncryptionSecret, f.created.ConsumerSecretEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "key-1", key)
	assert.Equal(t, "integrations.connection.connected", f.subject)
}

func TestConnectPostNL_seedsSettingsTrackingLinkAndStatusPolling(t *testing.T) {
	f := &postnlConnectFixture{}

	_, err := f.useCase(t).Execute(tenant.WithTenantID(context.Background(), "tenant-1"), validPostNLInput())

	require.NoError(t, err)
	require.NotNil(t, f.settings)
	assert.Equal(t, shippingDomain.CarrierPostNL, f.settings.DefaultCarrier)
	assert.Equal(t, shippingDomain.PackageTypePackage, f.settings.DefaultPackageType)
	assert.Equal(t, shippingDomain.LabelFormatPDF, f.settings.DefaultLabelFormat)

	require.NotNil(t, f.overrides.upserted)
	assert.Equal(t, shippingDomain.CarrierPostNL, f.overrides.upserted.Carrier)
	require.NotNil(t, f.overrides.upserted.TrackingURLTemplate)
	assert.Equal(t, "https://jouw.postnl.nl/track-and-trace/{barcode}-NL-{postcode}", *f.overrides.upserted.TrackingURLTemplate)

	s := f.schedules.upserted
	require.NotNil(t, s, "PostNL has no webhooks, so status comes only from polling")
	assert.Equal(t, "conn-postnl-1", s.ConnectionID)
	assert.Equal(t, channelsDomain.SyncEntityShipmentReconciliation, s.EntityType)
	assert.Equal(t, channelsDomain.SyncDirectionImport, s.Direction)
	assert.True(t, s.Enabled)
	assert.Equal(t, 60, s.IntervalMinutes)
	assert.NotNil(t, s.NextRunAt)
}

func TestConnectPostNL_rejectsInvalidAccount(t *testing.T) {
	cases := map[string]func(*appintegrations.ConnectPostNLInput){
		"no api key":               func(in *appintegrations.ConnectPostNLInput) { in.APIKey = " " },
		"no customer code":         func(in *appintegrations.ConnectPostNLInput) { in.CustomerCode = "" },
		"customer code too long":   func(in *appintegrations.ConnectPostNLInput) { in.CustomerCode = "ABCDE" },
		"no customer number":       func(in *appintegrations.ConnectPostNLInput) { in.CustomerNumber = "" },
		"customer number too long": func(in *appintegrations.ConnectPostNLInput) { in.CustomerNumber = "12345678901" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := &postnlConnectFixture{}
			in := validPostNLInput()
			mutate(&in)

			_, err := appintegrations.NewConnectPostNLUseCase(nil, nil, nil, nil, nil, nil, testEncryptionSecret).
				Execute(tenant.WithTenantID(context.Background(), "tenant-1"), in)

			assert.ErrorIs(t, err, integrationsDomain.ErrPostNLAccountInvalid)
			assert.Nil(t, f.created)
		})
	}
}

func TestConnectPostNL_requiresTenant(t *testing.T) {
	_, err := appintegrations.NewConnectPostNLUseCase(nil, nil, nil, nil, nil, nil, testEncryptionSecret).
		Execute(context.Background(), validPostNLInput())

	assert.ErrorIs(t, err, integrationsDomain.ErrNoTenantContext)
}
