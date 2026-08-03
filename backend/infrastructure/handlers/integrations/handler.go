package integrations

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	gencallback "github.com/adirkuhn/commerceos/api/integrationscallback"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type listConnectionsUC interface {
	Execute(ctx context.Context) ([]integrationsDomain.Connection, error)
}
type startWooConnectUC interface {
	Execute(ctx context.Context, in appintegrations.StartWooConnectInput) (*appintegrations.StartWooConnectResult, error)
}
type completeWooConnectUC interface {
	Execute(ctx context.Context, in appintegrations.CompleteWooConnectInput) (*integrationsDomain.Connection, error)
}
type manualConnectWooCommerceUC interface {
	Execute(ctx context.Context, in appintegrations.ManualConnectWooCommerceInput) (*integrationsDomain.Connection, error)
}
type disconnectConnectionUC interface {
	Execute(ctx context.Context, id string) error
}
type testConnectionUC interface {
	Execute(ctx context.Context, id string) (*integrationsDomain.Connection, error)
}
type updateConnectionSettingsUC interface {
	Execute(ctx context.Context, in appintegrations.UpdateConnectionSettingsInput) (*integrationsDomain.Connection, error)
}
type importWooProductsUC interface {
	Execute(ctx context.Context, in appchannels.ImportWooProductsInput) (*appchannels.ImportResult, error)
}
type pushProductToWooUC interface {
	Execute(ctx context.Context, in appchannels.PushProductToWooInput) (*channelsDomain.ProductChannelListing, error)
}
type listSyncSchedulesUC interface {
	Execute(ctx context.Context, connectionID string) ([]channelsDomain.SyncSchedule, error)
}
type deleteAllImportedProductsUC interface {
	Execute(ctx context.Context, in appchannels.DeleteAllImportedProductsInput) (*appchannels.DeleteAllImportedProductsResult, error)
}
type upsertSyncScheduleUC interface {
	Execute(ctx context.Context, in appchannels.UpsertSyncScheduleInput) (*channelsDomain.SyncSchedule, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listConnections          listConnectionsUC
	startWooConnect          startWooConnectUC
	completeWooConnect       completeWooConnectUC
	manualConnectWoo         manualConnectWooCommerceUC
	disconnect               disconnectConnectionUC
	testConnection           testConnectionUC
	importWooProducts        importWooProductsUC
	pushProductToWoo         pushProductToWooUC
	listSyncSchedules        listSyncSchedulesUC
	upsertSyncSchedule       upsertSyncScheduleUC
	deleteAllImportedProds   deleteAllImportedProductsUC
	updateConnectionSettings updateConnectionSettingsUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listConnections listConnectionsUC,
	startWooConnect startWooConnectUC,
	completeWooConnect completeWooConnectUC,
	manualConnectWoo manualConnectWooCommerceUC,
	disconnect disconnectConnectionUC,
	testConnection testConnectionUC,
	importWooProducts importWooProductsUC,
	pushProductToWoo pushProductToWooUC,
	listSyncSchedules listSyncSchedulesUC,
	upsertSyncSchedule upsertSyncScheduleUC,
	deleteAllImportedProds deleteAllImportedProductsUC,
	updateConnectionSettings updateConnectionSettingsUC,
) *Handler {
	return &Handler{
		listConnections:          listConnections,
		startWooConnect:          startWooConnect,
		completeWooConnect:       completeWooConnect,
		manualConnectWoo:         manualConnectWoo,
		disconnect:               disconnect,
		testConnection:           testConnection,
		importWooProducts:        importWooProducts,
		pushProductToWoo:         pushProductToWoo,
		listSyncSchedules:        listSyncSchedules,
		upsertSyncSchedule:       upsertSyncSchedule,
		deleteAllImportedProds:   deleteAllImportedProds,
		updateConnectionSettings: updateConnectionSettings,
	}
}

var _ genintegrations.ServerInterface = (*Handler)(nil)
var _ gencallback.ServerInterface = (*Handler)(nil)

// RegisterRoutes mounts the JWT-protected routes (matching the
// api/integrations OpenAPI spec exactly).
func RegisterRoutes(rg gin.IRouter, h *Handler) {
	genintegrations.RegisterHandlers(rg, h)
}

// RegisterPublicRoutes mounts the unauthenticated WooCommerce callback
// (matching the separate api/integrationscallback OpenAPI spec) — must be
// called on the root router, outside the JWT-protected group, since
// WooCommerce calls this server-to-server with no bearer token. BaseURL is
// set explicitly since, unlike the protected routes, this isn't mounted
// under a /api/v1 gin group that would supply the prefix automatically.
func RegisterPublicRoutes(r gin.IRouter, h *Handler) {
	gencallback.RegisterHandlersWithOptions(r, h, gencallback.GinServerOptions{BaseURL: "/api/v1"})
}

// Register is the production wiring: creates repos + use cases, then calls
// RegisterRoutes/RegisterPublicRoutes.
func Register(
	rg gin.IRouter,
	root gin.IRouter,
	db *gorm.DB,
	outboxRepo outbox.Repository,
	wooFactory integrationsDomain.WooClientFactory,
	stateSecret []byte,
	encryptionSecret, publicAPIBaseURL, frontendBaseURL string,
) {
	connRepo := repositories.NewConnectionRepo(db)
	categoryRepo := repositories.NewCategoryRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)
	stockMovementRepo := repositories.NewStockMovementRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)

	h := NewHandler(
		appintegrations.NewListConnectionsUseCase(db, connRepo),
		appintegrations.NewStartWooConnectUseCase(stateSecret, publicAPIBaseURL, frontendBaseURL),
		appintegrations.NewCompleteWooConnectUseCase(db, connRepo, outboxRepo, stateSecret, encryptionSecret),
		appintegrations.NewManualConnectWooCommerceUseCase(db, connRepo, outboxRepo, encryptionSecret),
		appintegrations.NewDisconnectConnectionUseCase(db, connRepo),
		appintegrations.NewTestConnectionUseCase(db, connRepo, wooFactory, encryptionSecret),
		appchannels.NewImportWooProductsUseCase(
			db, connRepo, wooFactory, encryptionSecret,
			categoryRepo, productRepo, variantRepo, skuRepo, listingRepo,
			appinventory.NewRecordStockMovementUseCase(db, stockMovementRepo),
		),
		appchannels.NewPushProductToWooUseCase(
			db, connRepo, wooFactory, encryptionSecret,
			productRepo, variantRepo, skuRepo, listingRepo,
			appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, repositories.NewBatchRepo(db)),
		),
		appchannels.NewListSyncSchedulesUseCase(db, scheduleRepo),
		appchannels.NewUpsertSyncScheduleUseCase(db, scheduleRepo),
		appchannels.NewDeleteAllImportedProductsUseCase(db, listingRepo, productRepo, categoryRepo),
		appintegrations.NewUpdateConnectionSettingsUseCase(db, connRepo),
	)
	RegisterRoutes(rg, h)
	RegisterPublicRoutes(root, h)
}

// ── Route handlers (api/integrations — protected) ────────────────────────────

func (h *Handler) ListCatalog(c *gin.Context) {
	middleware.OK(c, []genintegrations.CatalogEntry{
		{Provider: string(integrationsDomain.ProviderWooCommerce), Name: "WooCommerce"},
	})
}

func (h *Handler) ListConnections(c *gin.Context) {
	conns, err := h.listConnections.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.Connection, len(conns))
	for i, conn := range conns {
		resp[i] = toGenConnection(conn)
	}
	middleware.OK(c, resp)
}

func (h *Handler) StartWooConnect(c *gin.Context) {
	var req genintegrations.StartWooConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.startWooConnect.Execute(c.Request.Context(), appintegrations.StartWooConnectInput{StoreURL: req.StoreUrl})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"authorize_url": result.AuthorizeURL})
}

func (h *Handler) ManualConnectWooCommerce(c *gin.Context) {
	var req genintegrations.ManualConnectWooCommerceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.StoreUrl == "" || req.ConsumerKey == "" || req.ConsumerSecret == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "store_url, consumer_key, and consumer_secret are required")
		return
	}
	conn, err := h.manualConnectWoo.Execute(c.Request.Context(), appintegrations.ManualConnectWooCommerceInput{
		StoreURL:       req.StoreUrl,
		ConsumerKey:    req.ConsumerKey,
		ConsumerSecret: req.ConsumerSecret,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenConnection(*conn)
	middleware.OK(c, resp)
}

func (h *Handler) DeleteConnection(c *gin.Context, id genintegrations.Id) {
	err := h.disconnect.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) TestConnection(c *gin.Context, id genintegrations.Id) {
	conn, err := h.testConnection.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenConnection(*conn)
	middleware.OK(c, resp)
}

func (h *Handler) UpdateConnectionSettings(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpdateConnectionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	conn, err := h.updateConnectionSettings.Execute(c.Request.Context(), appintegrations.UpdateConnectionSettingsInput{
		ID: id, DefaultCurrency: req.DefaultCurrency,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) ImportWooProducts(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.ImportWooProductsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.DefaultWarehouseId == "" || req.DefaultCurrency == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "default_warehouse_id and default_currency are required")
		return
	}
	result, err := h.importWooProducts.Execute(c.Request.Context(), appchannels.ImportWooProductsInput{
		ConnectionID:       id,
		DefaultWarehouseID: req.DefaultWarehouseId,
		DefaultCurrency:    req.DefaultCurrency,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genintegrations.ImportResult{
		Imported:      result.Imported,
		AlreadyLinked: result.AlreadyLinked,
		SkippedType:   result.SkippedType,
		Errors:        emptySliceIfNil(result.Errors),
	})
}

func (h *Handler) PushProductToWoo(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.PushProductToWooRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.ProductId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "product_id is required")
		return
	}
	listing, err := h.pushProductToWoo.Execute(c.Request.Context(), appchannels.PushProductToWooInput{
		ProductID: req.ProductId, ConnectionID: id,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListing(*listing))
}

func (h *Handler) DeleteAllImportedProducts(c *gin.Context, id genintegrations.Id) {
	result, err := h.deleteAllImportedProds.Execute(c.Request.Context(), appchannels.DeleteAllImportedProductsInput{ConnectionID: id})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genintegrations.DeleteAllImportedProductsResult{
		DeletedProducts:   result.DeletedProducts,
		DeletedCategories: result.DeletedCategories,
		Errors:            emptySliceIfNil(result.Errors),
	})
}

func (h *Handler) ListSyncSchedules(c *gin.Context, id genintegrations.Id) {
	schedules, err := h.listSyncSchedules.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncSchedule, len(schedules))
	for i, s := range schedules {
		resp[i] = toGenSyncSchedule(s)
	}
	middleware.OK(c, resp)
}

// UpsertSyncSchedules saves each row via its own independent upsert
// (idempotent), rather than one all-or-nothing transaction spanning the
// whole batch. If a later row in the request fails validation, earlier
// rows in the same request have already been saved — safe to retry the
// whole PUT once the bad row is fixed, since re-applying already-saved
// rows is a no-op (same values in, same values out).
func (h *Handler) UpsertSyncSchedules(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpsertSyncSchedulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	resp := make([]genintegrations.SyncSchedule, 0, len(req.Schedules))
	for _, s := range req.Schedules {
		updated, err := h.upsertSyncSchedule.Execute(c.Request.Context(), appchannels.UpsertSyncScheduleInput{
			ConnectionID:       id,
			EntityType:         channelsDomain.SyncEntityType(s.EntityType),
			Direction:          channelsDomain.SyncDirection(s.Direction),
			Enabled:            s.Enabled,
			IntervalMinutes:    s.IntervalMinutes,
			DefaultWarehouseID: s.DefaultWarehouseId,
			DefaultCurrency:    s.DefaultCurrency,
		})
		if err := mapError(c, err); err != nil {
			return
		}
		resp = append(resp, toGenSyncSchedule(*updated))
	}
	middleware.OK(c, resp)
}

// ── Route handler (api/integrationscallback — public) ────────────────────────

// WooCallback is unauthenticated — WooCommerce itself calls this
// server-to-server after the merchant approves on their own store. Must
// return 200 quickly: WooCommerce deletes the newly created API key on any
// non-2xx response.
func (h *Handler) WooCallback(c *gin.Context) {
	var req gencallback.WooCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	conn, err := h.completeWooConnect.Execute(c.Request.Context(), appintegrations.CompleteWooConnectInput{
		State:          req.UserId,
		ConsumerKey:    req.ConsumerKey,
		ConsumerSecret: req.ConsumerSecret,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"connection_id": conn.ID})
}

// ── Domain → generated-type mapper ───────────────────────────────────────────

func toGenConnection(c integrationsDomain.Connection) genintegrations.Connection {
	currency := c.DefaultCurrency
	return genintegrations.Connection{
		Id:              c.ID,
		Provider:        string(c.Provider),
		StoreUrl:        c.StoreURL.String(),
		Status:          genintegrations.ConnectionStatus(c.Status),
		DefaultCurrency: &currency,
		CreatedAt:       c.CreatedAt,
	}
}

func toGenListing(l channelsDomain.ProductChannelListing) genintegrations.ProductChannelListing {
	return genintegrations.ProductChannelListing{
		Id:           l.ID,
		ProductId:    l.ProductID,
		ConnectionId: l.ConnectionID,
		Provider:     l.Provider,
		ExternalId:   l.ExternalID,
		SyncStatus:   genintegrations.ProductChannelListingSyncStatus(l.SyncStatus),
		LastSyncedAt: l.LastSyncedAt,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

func toGenSyncSchedule(s channelsDomain.SyncSchedule) genintegrations.SyncSchedule {
	return genintegrations.SyncSchedule{
		EntityType:         genintegrations.SyncScheduleEntityType(s.EntityType),
		Direction:          genintegrations.SyncScheduleDirection(s.Direction),
		Enabled:            s.Enabled,
		IntervalMinutes:    s.IntervalMinutes,
		DefaultWarehouseId: s.DefaultWarehouseID,
		DefaultCurrency:    s.DefaultCurrency,
		LastRunAt:          s.LastRunAt,
		LastRunStatus:      genintegrations.SyncScheduleLastRunStatus(s.LastRunStatus),
		LastRunError:       s.LastRunError,
	}
}

// emptySliceIfNil normalizes a nil slice to an empty one — a nil []string
// marshals to JSON `null`, but the OpenAPI schema declares errors as an
// array; frontend code that unconditionally reads `.errors.length` crashes
// on `null`. This keeps the wire contract's implied non-nullability true.
func emptySliceIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, integrationsDomain.ErrConnectionNotFound),
		errors.Is(err, pmsDomain.ErrProductNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, integrationsDomain.ErrStoreURLInvalid):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrProviderNotSupported):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrCannotPushVariableProduct),
		errors.Is(err, channelsDomain.ErrInvalidSyncEntityType),
		errors.Is(err, channelsDomain.ErrInvalidSyncDirection),
		errors.Is(err, channelsDomain.ErrInvalidSyncInterval),
		errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity),
		errors.Is(err, channelsDomain.ErrDefaultWarehouseAndCurrencyRequired),
		errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrStateInvalid):
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, integrationsDomain.ErrConnectionTestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrWooRequestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrNoTenantContext):
		middleware.Fail(c, http.StatusBadRequest, "NO_TENANT", err.Error())
	default:
		log.Printf("integrations: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
