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

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
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
	out *channelsDomain.ProductChannelListing
	err error
}

func (s *stubPushProductToWoo) Execute(_ context.Context, _ appchannels.PushProductToWooInput) (*channelsDomain.ProductChannelListing, error) {
	if s.out != nil {
		return s.out, s.err
	}
	return &channelsDomain.ProductChannelListing{}, s.err
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
	return &integrationsDomain.Connection{ID: in.ID, DefaultCurrency: in.DefaultCurrency}, nil
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
	return integrationshandler.NewHandler(list, start, complete, manual, disconnect, testConn,
		&stubCreateImportJob{}, &stubPushProductToWoo{}, &stubListSyncSchedules{}, &stubUpsertSyncSchedule{},
		&stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{})
}

// newSyncScheduleTestHandler builds a Handler for the sync-schedule routes
// specifically — every other use case is a harmless zero-value stub, same
// convention as newTestHandler's own hardcoded importWooProducts/pushProductToWoo.
func newSyncScheduleTestHandler(list *stubListSyncSchedules, upsert *stubUpsertSyncSchedule) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		list, upsert, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
	)
}

// newDeleteImportedTestHandler builds a Handler for the delete-all-imported
// route specifically — every other use case is a harmless zero-value stub.
func newDeleteImportedTestHandler(deleteAll *stubDeleteAllImportedProducts) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, deleteAll, &stubUpdateConnectionSettings{},
	)
}

// newUpdateConnectionSettingsTestHandler builds a Handler for the
// connection-settings route specifically.
func newUpdateConnectionSettingsTestHandler(settings *stubUpdateConnectionSettings) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, settings,
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

func TestCatalogHandler_returnsWooCommerceOnly(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/catalog", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "woocommerce")
	assert.NotContains(t, w.Body.String(), "shopify")
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
		&stubDisconnectConnection{}, &stubTestConnection{}, importUC, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
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

func TestUpdateConnectionSettingsHandler_notFound(t *testing.T) {
	h := newUpdateConnectionSettingsTestHandler(&stubUpdateConnectionSettings{err: integrationsDomain.ErrConnectionNotFound})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/connections/missing/settings", map[string]any{
		"default_currency": "USD",
	})

	assert.Equal(t, http.StatusNotFound, w.Code)
}
