package integrations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	integrationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
)

func init() { gin.SetMode(gin.TestMode) }

// ── Stub use cases ────────────────────────────────────────────────────────────

type stubListConnections struct {
	out []integrationsDomain.Connection
	err error
}

func (s *stubListConnections) Execute(_ context.Context) ([]integrationsDomain.Connection, error) {
	return s.out, s.err
}

type stubStartWooConnect struct {
	out *appintegrations.StartWooConnectResult
	err error
}

func (s *stubStartWooConnect) Execute(_ context.Context, _ appintegrations.StartWooConnectInput) (*appintegrations.StartWooConnectResult, error) {
	return s.out, s.err
}

type stubCompleteWooConnect struct {
	out *integrationsDomain.Connection
	err error
}

func (s *stubCompleteWooConnect) Execute(_ context.Context, _ appintegrations.CompleteWooConnectInput) (*integrationsDomain.Connection, error) {
	return s.out, s.err
}

type stubManualConnectWooCommerce struct {
	out *integrationsDomain.Connection
	err error
}

func (s *stubManualConnectWooCommerce) Execute(_ context.Context, _ appintegrations.ManualConnectWooCommerceInput) (*integrationsDomain.Connection, error) {
	return s.out, s.err
}

type stubDisconnectConnection struct{ err error }

func (s *stubDisconnectConnection) Execute(_ context.Context, _ string) error { return s.err }

type stubTestConnection struct {
	out *integrationsDomain.Connection
	err error
}

func (s *stubTestConnection) Execute(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
	return s.out, s.err
}

type stubListWooTaxes struct {
	out []appintegrations.WooTaxClassOption
	err error
}

func (s *stubListWooTaxes) Execute(_ context.Context, _ string) ([]appintegrations.WooTaxClassOption, error) {
	return s.out, s.err
}

type stubCreateImportJob struct {
	out *channelsDomain.ImportJob
	err error
}

func (s *stubCreateImportJob) Execute(_ context.Context, in appchannels.CreateImportJobInput) (*channelsDomain.ImportJob, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &channelsDomain.ImportJob{ID: "job-1", ConnectionID: in.ConnectionID, Status: channelsDomain.ImportJobPending}, nil
}

type stubPushProductToWoo struct {
	out []channelsDomain.ProductChannelListing
	err error
}

func (s *stubPushProductToWoo) Execute(_ context.Context, _ appchannels.PushProductToWooInput) ([]channelsDomain.ProductChannelListing, error) {
	if s.out != nil {
		return s.out, s.err
	}
	return []channelsDomain.ProductChannelListing{{}}, s.err
}

type stubListSyncSchedules struct {
	out []channelsDomain.SyncSchedule
	err error
}

func (s *stubListSyncSchedules) Execute(_ context.Context, _ string) ([]channelsDomain.SyncSchedule, error) {
	return s.out, s.err
}

type stubUpsertSyncSchedule struct {
	out *channelsDomain.SyncSchedule
	err error
}

func (s *stubUpsertSyncSchedule) Execute(_ context.Context, in appchannels.UpsertSyncScheduleInput) (*channelsDomain.SyncSchedule, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &channelsDomain.SyncSchedule{
		EntityType: in.EntityType, Direction: in.Direction, Enabled: in.Enabled, IntervalMinutes: in.IntervalMinutes,
	}, nil
}

type stubDeleteAllImportedProducts struct {
	out *appchannels.DeleteAllImportedProductsResult
	err error
}

func (s *stubDeleteAllImportedProducts) Execute(_ context.Context, _ appchannels.DeleteAllImportedProductsInput) (*appchannels.DeleteAllImportedProductsResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appchannels.DeleteAllImportedProductsResult{}, nil
}

type stubUpdateConnectionSettings struct {
	out *integrationsDomain.Connection
	err error
}

func (s *stubUpdateConnectionSettings) Execute(_ context.Context, in appintegrations.UpdateConnectionSettingsInput) (*integrationsDomain.Connection, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &integrationsDomain.Connection{ID: in.ID, DefaultCurrency: in.DefaultCurrency, DefaultWarehouseID: in.DefaultWarehouseID}, nil
}

type stubGetProductChannelListing struct {
	out *channelsDomain.ProductChannelListing
	err error
}

func (s *stubGetProductChannelListing) Execute(_ context.Context, _, _ string) (*channelsDomain.ProductChannelListing, error) {
	return s.out, s.err
}

type stubListSyncLogs struct {
	out []channelsDomain.SyncLogEntry
	err error
}

func (s *stubListSyncLogs) Execute(_ context.Context, _, _ string) ([]channelsDomain.SyncLogEntry, error) {
	return s.out, s.err
}

type stubListAllSyncLogs struct {
	out   []appchannels.SyncLogEntryWithNames
	total int64
	err   error
}

func (s *stubListAllSyncLogs) Execute(_ context.Context, _, _ int) ([]appchannels.SyncLogEntryWithNames, int64, error) {
	return s.out, s.total, s.err
}

type stubGetAIConnection struct {
	out *integrationsDomain.AIConnection
	err error
}

func (s *stubGetAIConnection) Execute(_ context.Context) (*integrationsDomain.AIConnection, error) {
	return s.out, s.err
}

type stubConnectAIProvider struct {
	out      *integrationsDomain.AIConnection
	err      error
	gotInput appintegrations.ConnectAIProviderInput
}

func (s *stubConnectAIProvider) Execute(_ context.Context, in appintegrations.ConnectAIProviderInput) (*integrationsDomain.AIConnection, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &integrationsDomain.AIConnection{ID: "ai-conn-1", Provider: in.Provider, Status: integrationsDomain.StatusConnected}, nil
}

type stubDisconnectAIConnection struct {
	err   error
	calls int
}

func (s *stubDisconnectAIConnection) Execute(_ context.Context) error {
	s.calls++
	return s.err
}

type stubListSalesChannels struct {
	out []channelsDomain.SalesChannel
	err error
}

func (s *stubListSalesChannels) Execute(_ context.Context) ([]channelsDomain.SalesChannel, error) {
	return s.out, s.err
}

type stubCreateSalesChannel struct {
	out *channelsDomain.SalesChannel
	err error
	in  appchannels.CreateSalesChannelInput
}

func (s *stubCreateSalesChannel) Execute(_ context.Context, in appchannels.CreateSalesChannelInput) (*channelsDomain.SalesChannel, error) {
	s.in = in
	return s.out, s.err
}

type stubUpdateSalesChannel struct {
	out *channelsDomain.SalesChannel
	err error
	in  appchannels.UpdateSalesChannelInput
}

func (s *stubUpdateSalesChannel) Execute(_ context.Context, in appchannels.UpdateSalesChannelInput) (*channelsDomain.SalesChannel, error) {
	s.in = in
	return s.out, s.err
}

type stubDeleteSalesChannel struct {
	err error
	id  string
}

func (s *stubDeleteSalesChannel) Execute(_ context.Context, id string) error {
	s.id = id
	return s.err
}

type stubListSalesChannelProducts struct {
	out   []appchannels.ProductEnrollment
	total int64
	err   error
}

func (s *stubListSalesChannelProducts) Execute(_ context.Context, _ appchannels.ListProductsWithChannelEnrollmentInput) ([]appchannels.ProductEnrollment, int64, error) {
	return s.out, s.total, s.err
}

type stubSetProductChannelOverride struct {
	out *channelsDomain.SalesChannelProductOverride
	err error
	in  appchannels.SetProductChannelOverrideInput
}

func (s *stubSetProductChannelOverride) Execute(_ context.Context, in appchannels.SetProductChannelOverrideInput) (*channelsDomain.SalesChannelProductOverride, error) {
	s.in = in
	return s.out, s.err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestHandler(
	list *stubListConnections,
	start *stubStartWooConnect,
	complete *stubCompleteWooConnect,
	manual *stubManualConnectWooCommerce,
	disconnect *stubDisconnectConnection,
	testConn *stubTestConnection,
) *integrationshandler.Handler {
	return integrationshandler.NewHandler(list, start, complete, manual, disconnect, testConn, &stubListWooTaxes{},
		&stubCreateImportJob{}, &stubPushProductToWoo{}, &stubListSyncSchedules{}, &stubUpsertSyncSchedule{},
		&stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
}

// newSyncScheduleTestHandler builds a Handler for the sync-schedule routes
// specifically — every other use case is a harmless zero-value stub, same
// convention as newTestHandler's own hardcoded importWooProducts/pushProductToWoo.
func newSyncScheduleTestHandler(list *stubListSyncSchedules, upsert *stubUpsertSyncSchedule) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		list, upsert, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newListWooTaxesTestHandler builds a Handler for the woo-taxes route
// specifically — every other use case is a harmless zero-value stub.
func newListWooTaxesTestHandler(listWooTaxes *stubListWooTaxes) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, listWooTaxes, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newDeleteImportedTestHandler builds a Handler for the delete-all-imported
// route specifically — every other use case is a harmless zero-value stub.
func newDeleteImportedTestHandler(deleteAll *stubDeleteAllImportedProducts) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, deleteAll, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newUpdateConnectionSettingsTestHandler builds a Handler for the
// connection-settings route specifically.
func newUpdateConnectionSettingsTestHandler(settings *stubUpdateConnectionSettings) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, settings, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newGetListingTestHandler builds a Handler for the get-product-channel-listing
// route specifically — every other use case is a harmless zero-value stub.
func newGetListingTestHandler(getListing *stubGetProductChannelListing) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		getListing, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newListSyncLogsTestHandler builds a Handler for the list-sync-logs route
// specifically — every other use case is a harmless zero-value stub.
func newListSyncLogsTestHandler(listLogs *stubListSyncLogs) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, listLogs, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newListAllSyncLogsTestHandler builds a Handler for the tenant-wide
// sync-logs route specifically — every other use case is a harmless
// zero-value stub.
func newListAllSyncLogsTestHandler(listAll *stubListAllSyncLogs) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, listAll,
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

func doRequest(rg *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCatalogHandler_returnsWooCommerceAndAI(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/catalog", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "woocommerce")
	assert.Contains(t, w.Body.String(), `"provider":"ai"`)
	assert.NotContains(t, w.Body.String(), "shopify")
}

func TestGetAIConnectionHandler_returnsNullWhenNotConfigured(t *testing.T) {
	h := newAIConnectionTestHandler(&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/ai", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":null`)
}

func TestGetAIConnectionHandler_returnsConfiguredConnection(t *testing.T) {
	get := &stubGetAIConnection{out: &integrationsDomain.AIConnection{
		ID: "ai-conn-1", Provider: integrationsDomain.AIProviderAnthropic, Status: integrationsDomain.StatusConnected,
	}}
	h := newAIConnectionTestHandler(get, &stubConnectAIProvider{}, &stubDisconnectAIConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/ai", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "anthropic")
}

func TestConnectAIProviderHandler_success(t *testing.T) {
	connect := &stubConnectAIProvider{}
	h := newAIConnectionTestHandler(&stubGetAIConnection{}, connect, &stubDisconnectAIConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai", map[string]any{
		"provider": "openai", "api_key": "sk-test", "model": "gpt-4o",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, integrationsDomain.AIProviderOpenAI, connect.gotInput.Provider)
	assert.Equal(t, "sk-test", connect.gotInput.APIKey)
	assert.Equal(t, "gpt-4o", connect.gotInput.Model)
}

func TestConnectAIProviderHandler_propagatesValidationError(t *testing.T) {
	connect := &stubConnectAIProvider{err: integrationsDomain.ErrAPIKeyRequired}
	h := newAIConnectionTestHandler(&stubGetAIConnection{}, connect, &stubDisconnectAIConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai", map[string]any{"provider": "anthropic"})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDisconnectAIProviderHandler_success(t *testing.T) {
	disconnect := &stubDisconnectAIConnection{}
	h := newAIConnectionTestHandler(&stubGetAIConnection{}, &stubConnectAIProvider{}, disconnect)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/ai", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, disconnect.calls)
}

func TestListConnectionsHandler_neverLeaksSecret(t *testing.T) {
	conns := []integrationsDomain.Connection{{
		ID:                      "conn-1",
		TenantID:                "tenant-1",
		Provider:                integrationsDomain.ProviderWooCommerce,
		StoreURL:                "https://mystore.com",
		ConsumerKey:             "ck_live_secret",
		ConsumerSecretEncrypted: "encrypted-blob",
		Status:                  integrationsDomain.StatusConnected,
		CreatedAt:               time.Now(),
	}}
	h := newTestHandler(&stubListConnections{out: conns}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mystore.com")
	assert.NotContains(t, w.Body.String(), "ck_live_secret")
	assert.NotContains(t, w.Body.String(), "encrypted-blob")
}

func TestStartWooConnectHandler_returnsAuthorizeURL(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{out: &appintegrations.StartWooConnectResult{AuthorizeURL: "https://mystore.com/wc-auth/v1/authorize?..."}}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/start", map[string]string{"store_url": "https://mystore.com"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "wc-auth/v1/authorize")
}

func TestStartWooConnectHandler_rejectsInvalidStoreURL(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{err: integrationsDomain.ErrStoreURLInvalid}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/start", map[string]string{"store_url": "not-a-url"})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDisconnectHandler_notFound(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{err: integrationsDomain.ErrConnectionNotFound}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/missing-id", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDisconnectHandler_success(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWooCallbackHandler_publicRoute_returns200OnSuccess(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Status: integrationsDomain.StatusConnected}
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{out: conn}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterPublicRoutes(r, h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", map[string]string{
		"key_id":          "1",
		"user_id":         "some-state-token",
		"consumer_key":    "ck_abc",
		"consumer_secret": "cs_abc",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWooCallbackHandler_invalidState_returns401(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{err: integrationsDomain.ErrStateInvalid}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterPublicRoutes(r, h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", map[string]string{
		"key_id":          "1",
		"user_id":         "bad-state",
		"consumer_key":    "ck_abc",
		"consumer_secret": "cs_abc",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWooCallbackHandler_isNotBehindJWTGroup(t *testing.T) {
	// RegisterPublicRoutes must be callable directly on the root router
	// (gin.IRouter satisfied by *gin.Engine), not require a RouterGroup with
	// JWT middleware already applied — this is the whole point of the split.
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{out: &integrationsDomain.Connection{}}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	engine := gin.New() // no middleware at all, proving no JWT is required
	integrationshandler.RegisterPublicRoutes(engine, h)

	w := doRequest(engine, http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", map[string]string{
		"key_id": "1", "user_id": "s", "consumer_key": "k", "consumer_secret": "v",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTestConnectionHandler_success(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Status: integrationsDomain.StatusConnected}
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{out: conn})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/test", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "connected")
}

func TestTestConnectionHandler_notFound(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{err: integrationsDomain.ErrConnectionNotFound})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/missing/test", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTestConnectionHandler_wooErrorReturns502WithUpdatedStatus(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Status: integrationsDomain.StatusError}
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{
		out: conn,
		err: fmt.Errorf("testConnection: %w: %w", integrationsDomain.ErrConnectionTestFailed, errors.New("401 unauthorized")),
	})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/test", nil)
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestManualConnectWooCommerceHandler_success(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", StoreURL: "https://mystore.com", Status: integrationsDomain.StatusConnected}
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{out: conn}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/manual", map[string]string{
		"store_url":       "https://mystore.com",
		"consumer_key":    "ck_abc",
		"consumer_secret": "cs_abc",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mystore.com")
	assert.NotContains(t, w.Body.String(), "ck_abc")
	assert.NotContains(t, w.Body.String(), "cs_abc")
}

func TestManualConnectWooCommerceHandler_rejectsMissingFields(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/manual", map[string]string{"store_url": "https://mystore.com"})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestManualConnectWooCommerceHandler_rejectsInvalidStoreURL(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{err: integrationsDomain.ErrStoreURLInvalid}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/manual", map[string]string{
		"store_url": "not-a-url", "consumer_key": "ck_abc", "consumer_secret": "cs_abc",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListSyncSchedulesHandler_returnsSchedules(t *testing.T) {
	warehouseID := "wh-1"
	h := newSyncScheduleTestHandler(&stubListSyncSchedules{out: []channelsDomain.SyncSchedule{
		{EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
			Enabled: true, IntervalMinutes: 15, DefaultWarehouseID: &warehouseID},
	}}, &stubUpsertSyncSchedule{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/sync-schedules", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "products")
	assert.Contains(t, w.Body.String(), "wh-1")
}

func TestListSyncSchedulesHandler_notFound(t *testing.T) {
	h := newSyncScheduleTestHandler(&stubListSyncSchedules{err: integrationsDomain.ErrConnectionNotFound}, &stubUpsertSyncSchedule{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/missing/sync-schedules", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListWooTaxesHandler_returnsDistinctTaxClasses(t *testing.T) {
	h := newListWooTaxesTestHandler(&stubListWooTaxes{out: []appintegrations.WooTaxClassOption{
		{Slug: "", Label: "Standard"},
		{Slug: "reduced-rate", Label: "Reduced Rate"},
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/woo-taxes", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Standard")
	assert.Contains(t, w.Body.String(), "reduced-rate")
}

func TestListWooTaxesHandler_notFound(t *testing.T) {
	h := newListWooTaxesTestHandler(&stubListWooTaxes{err: integrationsDomain.ErrConnectionNotFound})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/missing/woo-taxes", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListWooTaxesHandler_upstreamError(t *testing.T) {
	h := newListWooTaxesTestHandler(&stubListWooTaxes{err: integrationsDomain.ErrWooRequestFailed})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/woo-taxes", nil)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestUpsertSyncSchedulesHandler_savesEachRow(t *testing.T) {
	h := newSyncScheduleTestHandler(&stubListSyncSchedules{}, &stubUpsertSyncSchedule{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/connections/conn-1/sync-schedules", map[string]any{
		"schedules": []map[string]any{
			{"entity_type": "stock", "direction": "export", "enabled": true, "interval_minutes": 30, "last_run_status": "", "last_run_error": ""},
		},
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "stock")
}

func TestUpsertSyncSchedulesHandler_returns422ForUnrunnableEntity(t *testing.T) {
	h := newSyncScheduleTestHandler(&stubListSyncSchedules{}, &stubUpsertSyncSchedule{err: channelsDomain.ErrCannotEnableUnrunnableEntity})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/connections/conn-1/sync-schedules", map[string]any{
		"schedules": []map[string]any{
			{"entity_type": "orders", "direction": "import", "enabled": true, "interval_minutes": 30, "last_run_status": "", "last_run_error": ""},
		},
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteAllImportedProductsHandler_returnsResult(t *testing.T) {
	h := newDeleteImportedTestHandler(&stubDeleteAllImportedProducts{out: &appchannels.DeleteAllImportedProductsResult{
		DeletedProducts: 805, DeletedCategories: 12,
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/imported-products", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "805")
	assert.Contains(t, w.Body.String(), "12")
}

func TestDeleteAllImportedProductsHandler_returns500OnUnexpectedError(t *testing.T) {
	h := newDeleteImportedTestHandler(&stubDeleteAllImportedProducts{err: errors.New("db down")})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/imported-products", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// newImportTestHandler builds a Handler for the import route specifically —
// every other use case is a harmless zero-value stub.
func newImportTestHandler(importUC *stubCreateImportJob) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, importUC, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// newAIConnectionTestHandler builds a Handler for the AI connection routes
// specifically — every other use case is a harmless zero-value stub.
func newAIConnectionTestHandler(
	get *stubGetAIConnection, connect *stubConnectAIProvider, disconnect *stubDisconnectAIConnection,
) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		get, connect, disconnect,
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
	)
}

// The import endpoint no longer runs the import inline — it enqueues an
// ImportJob and returns immediately (202), so a large catalog no longer
// blocks the request (and the browser tab) for as long as the sync takes.
// cmd/scheduler picks up the pending job and runs it in the background.
func TestImportWooProductsHandler_returns202WithPendingJob(t *testing.T) {
	h := newImportTestHandler(&stubCreateImportJob{out: &channelsDomain.ImportJob{
		ID: "job-1", ConnectionID: "conn-1", Status: channelsDomain.ImportJobPending,
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/import", map[string]any{
		"default_warehouse_id": "wh-1", "default_currency": "EUR",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"pending"`)
	assert.Contains(t, w.Body.String(), `"id":"job-1"`)
}

// A nil []string (the zero value, e.g. when a job has no per-product
// errors yet) marshals to JSON `null`. Frontend code reads `.errors.length`
// unconditionally, assuming an array — `null` crashed it. The handler must
// normalize nil to [] so the wire contract's implied non-nullability holds.
func TestImportWooProductsHandler_normalizesNilErrorsToEmptyArray(t *testing.T) {
	h := newImportTestHandler(&stubCreateImportJob{out: &channelsDomain.ImportJob{ID: "job-1", Status: channelsDomain.ImportJobPending}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/import", map[string]any{
		"default_warehouse_id": "wh-1", "default_currency": "EUR",
	})

	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), `"errors":[]`)
	assert.NotContains(t, w.Body.String(), `"errors":null`)
}

func TestImportWooProductsHandler_returns409WhenAlreadyRunning(t *testing.T) {
	h := newImportTestHandler(&stubCreateImportJob{err: channelsDomain.ErrImportJobAlreadyActive})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/import", map[string]any{
		"default_warehouse_id": "wh-1", "default_currency": "EUR",
	})

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "IMPORT_ALREADY_RUNNING")
}

func TestDeleteAllImportedProductsHandler_normalizesNilErrorsToEmptyArray(t *testing.T) {
	h := newDeleteImportedTestHandler(&stubDeleteAllImportedProducts{out: &appchannels.DeleteAllImportedProductsResult{
		DeletedProducts: 5,
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/imported-products", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"errors":[]`)
	assert.NotContains(t, w.Body.String(), `"errors":null`)
}

func TestUpdateConnectionSettingsHandler_savesDefaultCurrency(t *testing.T) {
	h := newUpdateConnectionSettingsTestHandler(&stubUpdateConnectionSettings{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/connections/conn-1/settings", map[string]any{
		"default_currency": "USD",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "USD")
}

func TestUpdateConnectionSettingsHandler_savesDefaultWarehouse(t *testing.T) {
	h := newUpdateConnectionSettingsTestHandler(&stubUpdateConnectionSettings{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/connections/conn-1/settings", map[string]any{
		"default_currency":     "USD",
		"default_warehouse_id": "wh-1",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "wh-1")
}

func TestUpdateConnectionSettingsHandler_notFound(t *testing.T) {
	h := newUpdateConnectionSettingsTestHandler(&stubUpdateConnectionSettings{err: integrationsDomain.ErrConnectionNotFound})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/connections/missing/settings", map[string]any{
		"default_currency": "USD",
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetProductChannelListingHandler_returnsTheListing(t *testing.T) {
	h := newGetListingTestHandler(&stubGetProductChannelListing{out: &channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "woocommerce",
		ExternalID: "23609", SyncStatus: channelsDomain.SyncStatusSynced,
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/listings/prod-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "23609")
	assert.Contains(t, w.Body.String(), "synced")
}

func TestGetProductChannelListingHandler_notFoundWhenProductNeverLinked(t *testing.T) {
	h := newGetListingTestHandler(&stubGetProductChannelListing{err: channelsDomain.ErrListingNotFound})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/listings/prod-1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListSyncLogsHandler_returnsEntries(t *testing.T) {
	h := newListSyncLogsTestHandler(&stubListSyncLogs{out: []channelsDomain.SyncLogEntry{
		{
			ID: "log-1", ListingID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1",
			Provider: "woocommerce", ExternalID: "23609",
			Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusSuccess,
			Fields:  map[string]string{"name": "T-Shirt"},
			Changes: map[string]channelsDomain.FieldChange{"name": {From: "", To: "T-Shirt"}},
		},
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/listings/prod-1/sync-logs", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "log-1")
	assert.Contains(t, w.Body.String(), "T-Shirt")
	assert.Contains(t, w.Body.String(), `"push"`)
}

func TestListSyncLogsHandler_notFoundWhenListingMissing(t *testing.T) {
	h := newListSyncLogsTestHandler(&stubListSyncLogs{err: channelsDomain.ErrListingNotFound})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/listings/prod-1/sync-logs", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListAllSyncLogsHandler_returnsEntriesWithTotal(t *testing.T) {
	h := newListAllSyncLogsTestHandler(&stubListAllSyncLogs{
		out: []appchannels.SyncLogEntryWithNames{
			{
				SyncLogEntry: channelsDomain.SyncLogEntry{
					ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "woocommerce",
					Direction: channelsDomain.SyncFlowPush, Status: channelsDomain.SyncLogStatusSuccess,
				},
				ProductName: "T-Shirt", StoreURL: "https://mystore.com",
			},
		},
		total: 42,
	})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/sync-logs?page=2&per_page=10", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "T-Shirt")
	assert.Contains(t, w.Body.String(), "mystore.com")
	assert.Contains(t, w.Body.String(), `"total":42`)
}

func TestListAllSyncLogsHandler_defaultsPageAndPerPage(t *testing.T) {
	stub := &stubListAllSyncLogs{}
	h := newListAllSyncLogsTestHandler(stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/sync-logs", nil)

	require.Equal(t, http.StatusOK, w.Code)
}

// newSalesChannelTestHandler builds a Handler for the sales-channel routes
// specifically — every other use case is a harmless zero-value stub.
func newSalesChannelTestHandler(
	list *stubListSalesChannels,
	create *stubCreateSalesChannel,
	update *stubUpdateSalesChannel,
	del *stubDeleteSalesChannel,
	listProducts *stubListSalesChannelProducts,
	setOverride *stubSetProductChannelOverride,
) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		list, create, update, del, listProducts, setOverride,
	)
}

func TestListSalesChannelsHandler_returnsChannels(t *testing.T) {
	h := newSalesChannelTestHandler(&stubListSalesChannels{out: []channelsDomain.SalesChannel{
		{ID: "chan-1", Name: "My Woo Store", Type: channelsDomain.ChannelTypeWooCommerce, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/sales-channels", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "My Woo Store")
}

func TestCreateSalesChannelHandler_returns201WithChannel(t *testing.T) {
	create := &stubCreateSalesChannel{out: &channelsDomain.SalesChannel{
		ID: "chan-1", Name: "My Woo Store", Type: channelsDomain.ChannelTypeWooCommerce, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	}}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, create, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	body := genintegrations.CreateSalesChannelRequest{
		Name: "My Woo Store", Type: genintegrations.ChannelType(channelsDomain.ChannelTypeWooCommerce),
		EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll),
	}
	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sales-channels", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "My Woo Store")
	assert.Equal(t, channelsDomain.EnrollmentModeAll, create.in.EnrollmentMode)
}

func TestCreateSalesChannelHandler_returns422OnInvalidEnrollmentMode(t *testing.T) {
	create := &stubCreateSalesChannel{err: channelsDomain.ErrInvalidEnrollmentMode}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, create, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	body := genintegrations.CreateSalesChannelRequest{Name: "Bad", Type: genintegrations.ChannelType(channelsDomain.ChannelTypeWooCommerce), EnrollmentMode: "bogus"}
	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sales-channels", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateSalesChannelHandler_returnsUpdatedChannel(t *testing.T) {
	update := &stubUpdateSalesChannel{out: &channelsDomain.SalesChannel{ID: "chan-1", Name: "Renamed", EnrollmentMode: channelsDomain.EnrollmentModeAll}}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, &stubCreateSalesChannel{}, update, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	body := genintegrations.UpdateSalesChannelRequest{Name: "Renamed", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll), Active: true}
	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/sales-channels/chan-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Renamed")
	assert.Equal(t, "chan-1", update.in.ID)
}

func TestUpdateSalesChannelHandler_returns404WhenChannelNotFound(t *testing.T) {
	update := &stubUpdateSalesChannel{err: channelsDomain.ErrSalesChannelNotFound}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, &stubCreateSalesChannel{}, update, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	body := genintegrations.UpdateSalesChannelRequest{Name: "Renamed", EnrollmentMode: genintegrations.EnrollmentMode(channelsDomain.EnrollmentModeAll)}
	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/sales-channels/missing", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteSalesChannelHandler_returns200(t *testing.T) {
	del := &stubDeleteSalesChannel{}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, del, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/sales-channels/chan-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "chan-1", del.id)
}

func TestListSalesChannelProductsHandler_returnsProductsWithEnrollment(t *testing.T) {
	category := "cat-1"
	listProducts := &stubListSalesChannelProducts{
		out: []appchannels.ProductEnrollment{
			{Product: pmsDomain.Product{ID: "p1", Name: "T-Shirt", Slug: "t-shirt", CategoryID: &category, Tags: []string{"summer"}}, Enrolled: true},
		},
		total: 1,
	}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, listProducts, &stubSetProductChannelOverride{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/sales-channels/chan-1/products?page=1&per_page=20", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "T-Shirt")
	assert.Contains(t, w.Body.String(), `"enrolled":true`)
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestSetSalesChannelProductOverrideHandler_returns200(t *testing.T) {
	setOverride := &stubSetProductChannelOverride{out: &channelsDomain.SalesChannelProductOverride{}}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, setOverride)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	body := genintegrations.SetSalesChannelProductOverrideRequest{Included: true}
	w := doRequest(r, http.MethodPut, "/api/v1/integrations/sales-channels/chan-1/products/p1/override", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, setOverride.in.Included)
	assert.Equal(t, "chan-1", setOverride.in.ChannelID)
	assert.Equal(t, "p1", setOverride.in.ProductID)
}

func TestSetSalesChannelProductOverrideHandler_returns404WhenProductNotFound(t *testing.T) {
	setOverride := &stubSetProductChannelOverride{err: pmsDomain.ErrProductNotFound}
	h := newSalesChannelTestHandler(&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, setOverride)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	body := genintegrations.SetSalesChannelProductOverrideRequest{Included: true}
	w := doRequest(r, http.MethodPut, "/api/v1/integrations/sales-channels/chan-1/products/missing/override", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
