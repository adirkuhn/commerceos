package webhooks_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

const testEncryptionSecret = "test-encryption-secret"

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// however many tenant.WithTenant Begin/set_config/Commit sequences the test
// needs -- same pattern as application/channels' use-case tests.
func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

type fakeConnRepo struct {
	conns []integrationsDomain.Connection
}

func (f *fakeConnRepo) Create(ctx context.Context, c *integrationsDomain.Connection) error {
	return nil
}
func (f *fakeConnRepo) FindByID(ctx context.Context, id string) (*integrationsDomain.Connection, error) {
	return nil, nil
}
func (f *fakeConnRepo) ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error) {
	return f.conns, nil
}
func (f *fakeConnRepo) Update(ctx context.Context, c *integrationsDomain.Connection) error {
	return nil
}
func (f *fakeConnRepo) FindByStore(context.Context, integrationsDomain.Provider, integrationsDomain.StoreURL) (*integrationsDomain.Connection, error) {
	return nil, integrationsDomain.ErrConnectionNotFound
}

type fakePublisher struct {
	failCount   int
	failErr     error // returned instead of the generic "no responders" error when set
	calls       int
	lastSubject string
	lastData    []byte
}

func (f *fakePublisher) Publish(subject string, data []byte) error {
	f.calls++
	f.lastSubject, f.lastData = subject, data
	if f.calls <= f.failCount {
		if f.failErr != nil {
			return f.failErr
		}
		return errors.New("nats: no responders")
	}
	return nil
}

type fakeFallbackRepo struct {
	saved []webhooks.FallbackEvent
}

func (f *fakeFallbackRepo) Save(ctx context.Context, e *webhooks.FallbackEvent) error {
	f.saved = append(f.saved, *e)
	return nil
}
func (f *fakeFallbackRepo) Get(ctx context.Context, id string) (*webhooks.FallbackEvent, error) {
	return nil, nil
}
func (f *fakeFallbackRepo) List(ctx context.Context, status webhooks.FallbackStatus, limit, offset int) ([]webhooks.FallbackEvent, int64, error) {
	return nil, 0, nil
}
func (f *fakeFallbackRepo) UpdateStatusForAll(ctx context.Context, fromStatus, toStatus webhooks.FallbackStatus) (int64, error) {
	return 0, nil
}
func (f *fakeFallbackRepo) UpdateStatus(ctx context.Context, id string, status webhooks.FallbackStatus) error {
	return nil
}

func wooConnWithSecret(t *testing.T, secret string) integrationsDomain.Connection {
	t.Helper()
	enc, err := crypto.Encrypt(testEncryptionSecret, secret)
	require.NoError(t, err)
	return integrationsDomain.Connection{
		ID: "conn-1", TenantID: "tenant-1", Provider: integrationsDomain.ProviderWooCommerce,
		WebhookSecretEncrypted: enc,
	}
}

func wooSignature(t *testing.T, secret string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestReceiveWebhookUseCase_publishesOnValidSignature(t *testing.T) {
	secret := "shhh"
	connRepo := &fakeConnRepo{conns: []integrationsDomain.Connection{wooConnWithSecret(t, secret)}}
	pub := &fakePublisher{}
	fallbackRepo := &fakeFallbackRepo{}
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 1), connRepo, webhookproviders.NewRegistry(), pub, fallbackRepo, testEncryptionSecret)

	body := []byte(`{"id":123}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": wooSignature(t, secret, body)}

	result, err := uc.Execute(context.Background(), "tenant-1", "woocommerce", body, headers)

	require.NoError(t, err)
	assert.NotEmpty(t, result.EventID)
	assert.False(t, result.FallbackUsed)
	assert.Equal(t, 1, pub.calls)
	assert.Empty(t, fallbackRepo.saved)
}

func TestReceiveWebhookUseCase_rejectsInvalidSignature(t *testing.T) {
	connRepo := &fakeConnRepo{conns: []integrationsDomain.Connection{wooConnWithSecret(t, "real-secret")}}
	pub := &fakePublisher{}
	fallbackRepo := &fakeFallbackRepo{}
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 1), connRepo, webhookproviders.NewRegistry(), pub, fallbackRepo, testEncryptionSecret)

	body := []byte(`{"id":123}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": wooSignature(t, "wrong-secret", body)}

	_, err := uc.Execute(context.Background(), "tenant-1", "woocommerce", body, headers)

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
	assert.Zero(t, pub.calls)
}

func TestReceiveWebhookUseCase_unknownProvider404(t *testing.T) {
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 0), &fakeConnRepo{}, webhookproviders.NewRegistry(), &fakePublisher{}, &fakeFallbackRepo{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), "tenant-1", "totally-unknown-vendor", []byte("{}"), nil)

	assert.ErrorIs(t, err, webhooks.ErrProviderUnknown)
}

func TestReceiveWebhookUseCase_stubProviderNotImplemented(t *testing.T) {
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 1), &fakeConnRepo{}, webhookproviders.NewRegistry(), &fakePublisher{}, &fakeFallbackRepo{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), "tenant-1", "shopify", []byte("{}"), nil)

	assert.ErrorIs(t, err, webhooks.ErrProviderNotImplemented)
}

func TestReceiveWebhookUseCase_noConnectionConfigured501(t *testing.T) {
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 1), &fakeConnRepo{}, webhookproviders.NewRegistry(), &fakePublisher{}, &fakeFallbackRepo{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), "tenant-1", "woocommerce", []byte("{}"), nil)

	assert.ErrorIs(t, err, webhooks.ErrProviderNotImplemented)
}

func TestReceiveWebhookUseCase_retriesPublishThenFallsBackToDB(t *testing.T) {
	secret := "shhh"
	connRepo := &fakeConnRepo{conns: []integrationsDomain.Connection{wooConnWithSecret(t, secret)}}
	pub := &fakePublisher{failCount: 3} // fails all 3 allowed attempts
	fallbackRepo := &fakeFallbackRepo{}
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 2), connRepo, webhookproviders.NewRegistry(), pub, fallbackRepo, testEncryptionSecret)

	body := []byte(`{"id":123}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": wooSignature(t, secret, body)}

	result, err := uc.Execute(context.Background(), "tenant-1", "woocommerce", body, headers)

	require.NoError(t, err)
	assert.True(t, result.FallbackUsed)
	assert.Equal(t, 3, pub.calls)
	require.Len(t, fallbackRepo.saved, 1)
	assert.Equal(t, "tenant-1", fallbackRepo.saved[0].TenantID)
	assert.Equal(t, webhooks.FallbackStatusPendingRepublish, fallbackRepo.saved[0].Status)
	assert.Equal(t, body, fallbackRepo.saved[0].Payload)
}

// TestReceiveWebhookUseCase_payloadTooLarge_singleAttemptFallsBackToDB pins
// that a too-large payload isn't retried maxPublishRetries times -- every
// attempt would fail identically against the same oversized bytes, so
// retrying just delays the fallback save.
func TestReceiveWebhookUseCase_payloadTooLarge_singleAttemptFallsBackToDB(t *testing.T) {
	secret := "shhh"
	connRepo := &fakeConnRepo{conns: []integrationsDomain.Connection{wooConnWithSecret(t, secret)}}
	pub := &fakePublisher{failCount: 99, failErr: webhooks.ErrPayloadTooLarge}
	fallbackRepo := &fakeFallbackRepo{}
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 2), connRepo, webhookproviders.NewRegistry(), pub, fallbackRepo, testEncryptionSecret)

	body := []byte(`{"id":123}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": wooSignature(t, secret, body)}

	result, err := uc.Execute(context.Background(), "tenant-1", "woocommerce", body, headers)

	require.NoError(t, err)
	assert.True(t, result.FallbackUsed)
	assert.Equal(t, 1, pub.calls, "must not retry a deterministically-failing payload")
	require.Len(t, fallbackRepo.saved, 1)
}

func TestReceiveWebhookUseCase_publishSucceedsOnSecondAttempt(t *testing.T) {
	secret := "shhh"
	connRepo := &fakeConnRepo{conns: []integrationsDomain.Connection{wooConnWithSecret(t, secret)}}
	pub := &fakePublisher{failCount: 1}
	fallbackRepo := &fakeFallbackRepo{}
	uc := appwebhooks.NewReceiveWebhookUseCase(newMockGormDB(t, 1), connRepo, webhookproviders.NewRegistry(), pub, fallbackRepo, testEncryptionSecret)

	body := []byte(`{"id":123}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": wooSignature(t, secret, body)}

	result, err := uc.Execute(context.Background(), "tenant-1", "woocommerce", body, headers)

	require.NoError(t, err)
	assert.False(t, result.FallbackUsed)
	assert.Equal(t, 2, pub.calls)
	assert.Empty(t, fallbackRepo.saved)
}
