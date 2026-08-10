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
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
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
type createImportJobUC interface {
	Execute(ctx context.Context, in appchannels.CreateImportJobInput) (*channelsDomain.ImportJob, error)
}
type pushProductToWooUC interface {
	Execute(ctx context.Context, in appchannels.PushProductToWooInput) ([]channelsDomain.ProductChannelListing, error)
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
type getProductChannelListingUC interface {
	Execute(ctx context.Context, productID, connectionID string) (*channelsDomain.ProductChannelListing, error)
}
type listSyncLogsUC interface {
	Execute(ctx context.Context, productID, connectionID string) ([]channelsDomain.SyncLogEntry, error)
}
type listAllSyncLogsUC interface {
	Execute(ctx context.Context, page, perPage int) ([]appchannels.SyncLogEntryWithNames, int64, error)
}
type getAIConnectionUC interface {
	Execute(ctx context.Context) (*integrationsDomain.AIConnection, error)
}
type connectAIProviderUC interface {
	Execute(ctx context.Context, in appintegrations.ConnectAIProviderInput) (*integrationsDomain.AIConnection, error)
}
type disconnectAIConnectionUC interface {
	Execute(ctx context.Context) error
}
type listSalesChannelsUC interface {
	Execute(ctx context.Context) ([]channelsDomain.SalesChannel, error)
}
type createSalesChannelUC interface {
	Execute(ctx context.Context, in appchannels.CreateSalesChannelInput) (*channelsDomain.SalesChannel, error)
}
type updateSalesChannelUC interface {
	Execute(ctx context.Context, in appchannels.UpdateSalesChannelInput) (*channelsDomain.SalesChannel, error)
}
type deleteSalesChannelUC interface {
	Execute(ctx context.Context, id string) error
}
type listSalesChannelProductsUC interface {
	Execute(ctx context.Context, in appchannels.ListProductsWithChannelEnrollmentInput) ([]appchannels.ProductEnrollment, int64, error)
}
type setProductChannelOverrideUC interface {
	Execute(ctx context.Context, in appchannels.SetProductChannelOverrideInput) (*channelsDomain.SalesChannelProductOverride, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listConnections           listConnectionsUC
	startWooConnect           startWooConnectUC
	completeWooConnect        completeWooConnectUC
	manualConnectWoo          manualConnectWooCommerceUC
	disconnect                disconnectConnectionUC
	testConnection            testConnectionUC
	createImportJob           createImportJobUC
	pushProductToWoo          pushProductToWooUC
	listSyncSchedules         listSyncSchedulesUC
	upsertSyncSchedule        upsertSyncScheduleUC
	deleteAllImportedProds    deleteAllImportedProductsUC
	updateConnectionSettings  updateConnectionSettingsUC
	getProductChannelListing  getProductChannelListingUC
	listSyncLogs              listSyncLogsUC
	listAllSyncLogs           listAllSyncLogsUC
	getAIConnection           getAIConnectionUC
	connectAIProvider         connectAIProviderUC
	disconnectAIConnection    disconnectAIConnectionUC
	listSalesChannels         listSalesChannelsUC
	createSalesChannel        createSalesChannelUC
	updateSalesChannel        updateSalesChannelUC
	deleteSalesChannel        deleteSalesChannelUC
	listSalesChannelProducts  listSalesChannelProductsUC
	setProductChannelOverride setProductChannelOverrideUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listConnections listConnectionsUC,
	startWooConnect startWooConnectUC,
	completeWooConnect completeWooConnectUC,
	manualConnectWoo manualConnectWooCommerceUC,
	disconnect disconnectConnectionUC,
	testConnection testConnectionUC,
	createImportJob createImportJobUC,
	pushProductToWoo pushProductToWooUC,
	listSyncSchedules listSyncSchedulesUC,
	upsertSyncSchedule upsertSyncScheduleUC,
	deleteAllImportedProds deleteAllImportedProductsUC,
	updateConnectionSettings updateConnectionSettingsUC,
	getProductChannelListing getProductChannelListingUC,
	listSyncLogs listSyncLogsUC,
	listAllSyncLogs listAllSyncLogsUC,
	getAIConnection getAIConnectionUC,
	connectAIProvider connectAIProviderUC,
	disconnectAIConnection disconnectAIConnectionUC,
	listSalesChannels listSalesChannelsUC,
	createSalesChannel createSalesChannelUC,
	updateSalesChannel updateSalesChannelUC,
	deleteSalesChannel deleteSalesChannelUC,
	listSalesChannelProducts listSalesChannelProductsUC,
	setProductChannelOverride setProductChannelOverrideUC,
) *Handler {
	return &Handler{
		listConnections:           listConnections,
		startWooConnect:           startWooConnect,
		completeWooConnect:        completeWooConnect,
		manualConnectWoo:          manualConnectWoo,
		disconnect:                disconnect,
		testConnection:            testConnection,
		createImportJob:           createImportJob,
		pushProductToWoo:          pushProductToWoo,
		listSyncSchedules:         listSyncSchedules,
		upsertSyncSchedule:        upsertSyncSchedule,
		deleteAllImportedProds:    deleteAllImportedProds,
		updateConnectionSettings:  updateConnectionSettings,
		getProductChannelListing:  getProductChannelListing,
		listSyncLogs:              listSyncLogs,
		listAllSyncLogs:           listAllSyncLogs,
		getAIConnection:           getAIConnection,
		connectAIProvider:         connectAIProvider,
		disconnectAIConnection:    disconnectAIConnection,
		listSalesChannels:         listSalesChannels,
		createSalesChannel:        createSalesChannel,
		updateSalesChannel:        updateSalesChannel,
		deleteSalesChannel:        deleteSalesChannel,
		listSalesChannelProducts:  listSalesChannelProducts,
		setProductChannelOverride: setProductChannelOverride,
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
	syncLogRepo := repositories.NewSyncLogRepo(db)
	stockMovementRepo := repositories.NewStockMovementRepo(db)
	scheduleRepo := repositories.NewSyncScheduleRepo(db)
	importJobRepo := repositories.NewImportJobRepo(db)
	notificationRepo := repositories.NewNotificationRepo(db)
	aiConnRepo := repositories.NewAIConnectionRepo(db)
	salesChannelRepo := repositories.NewSalesChannelRepo(db)
	salesChannelOverrideRepo := repositories.NewSalesChannelProductOverrideRepo(db)

	h := NewHandler(
		appintegrations.NewListConnectionsUseCase(db, connRepo),
		appintegrations.NewStartWooConnectUseCase(stateSecret, publicAPIBaseURL, frontendBaseURL),
		appintegrations.NewCompleteWooConnectUseCase(db, connRepo, outboxRepo, stateSecret, encryptionSecret),
		appintegrations.NewManualConnectWooCommerceUseCase(db, connRepo, outboxRepo, encryptionSecret),
		appintegrations.NewDisconnectConnectionUseCase(db, connRepo),
		appintegrations.NewTestConnectionUseCase(db, connRepo, wooFactory, encryptionSecret),
		appchannels.NewCreateImportJobUseCase(
			db, importJobRepo, appnotifications.NewCreateNotificationUseCase(db, notificationRepo),
		),
		appchannels.NewPushProductToWooUseCase(
			db, connRepo, wooFactory, encryptionSecret,
			productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo,
			appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, repositories.NewBatchRepo(db)),
		),
		appchannels.NewListSyncSchedulesUseCase(db, scheduleRepo),
		appchannels.NewUpsertSyncScheduleUseCase(db, scheduleRepo),
		appchannels.NewDeleteAllImportedProductsUseCase(db, listingRepo, productRepo, categoryRepo),
		appintegrations.NewUpdateConnectionSettingsUseCase(db, connRepo),
		appchannels.NewGetProductChannelListingUseCase(db, listingRepo),
		appchannels.NewListSyncLogsUseCase(db, listingRepo, syncLogRepo),
		appchannels.NewListAllSyncLogsUseCase(db, syncLogRepo, productRepo, connRepo),
		appintegrations.NewGetAIConnectionUseCase(db, aiConnRepo),
		appintegrations.NewConnectAIProviderUseCase(db, aiConnRepo, encryptionSecret),
		appintegrations.NewDisconnectAIConnectionUseCase(db, aiConnRepo),
		appchannels.NewListSalesChannelsUseCase(db, salesChannelRepo),
		appchannels.NewCreateSalesChannelUseCase(db, salesChannelRepo, connRepo),
		appchannels.NewUpdateSalesChannelUseCase(db, salesChannelRepo, connRepo),
		appchannels.NewDeleteSalesChannelUseCase(db, salesChannelRepo),
		appchannels.NewListProductsWithChannelEnrollmentUseCase(db, productRepo, salesChannelRepo, salesChannelOverrideRepo),
		appchannels.NewSetProductChannelOverrideUseCase(db, salesChannelOverrideRepo, salesChannelRepo, productRepo),
	)
	RegisterRoutes(rg, h)
	RegisterPublicRoutes(root, h)
}

// ── Route handlers (api/integrations — protected) ────────────────────────────

func (h *Handler) ListCatalog(c *gin.Context) {
	middleware.OK(c, []genintegrations.CatalogEntry{
		{Provider: string(integrationsDomain.ProviderWooCommerce), Name: "WooCommerce"},
		{Provider: "ai", Name: "AI"},
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

func (h *Handler) GetAIConnection(c *gin.Context) {
	conn, err := h.getAIConnection.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	if conn == nil {
		middleware.OK(c, nil)
		return
	}
	resp := toGenAIConnection(*conn)
	middleware.OK(c, &resp)
}

func (h *Handler) ConnectAIProvider(c *gin.Context) {
	var req genintegrations.ConnectAIProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	in := appintegrations.ConnectAIProviderInput{Provider: integrationsDomain.AIProvider(req.Provider)}
	if req.ApiKey != nil {
		in.APIKey = *req.ApiKey
	}
	if req.BaseUrl != nil {
		in.BaseURL = *req.BaseUrl
	}
	if req.Model != nil {
		in.Model = *req.Model
	}
	conn, err := h.connectAIProvider.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenAIConnection(*conn)
	middleware.OK(c, &resp)
}

func (h *Handler) DisconnectAIProvider(c *gin.Context) {
	if err := mapError(c, h.disconnectAIConnection.Execute(c.Request.Context())); err != nil {
		return
	}
	middleware.OK(c, nil)
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
	job, err := h.createImportJob.Execute(c.Request.Context(), appchannels.CreateImportJobInput{
		ConnectionID:       id,
		DefaultWarehouseID: req.DefaultWarehouseId,
		DefaultCurrency:    req.DefaultCurrency,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusAccepted, middleware.APIResponse{Data: toGenImportJob(*job)})
}

func toGenImportJob(j channelsDomain.ImportJob) genintegrations.ImportJob {
	return genintegrations.ImportJob{
		Id: j.ID, ConnectionId: j.ConnectionID,
		Status:        genintegrations.ImportJobStatus(j.Status),
		Imported:      j.Imported,
		AlreadyLinked: j.AlreadyLinked,
		SkippedType:   j.SkippedType,
		Errors:        emptySliceIfNil(j.Errors),
		ErrorMessage:  j.ErrorMessage,
		CreatedAt:     j.CreatedAt,
		StartedAt:     j.StartedAt,
		FinishedAt:    j.FinishedAt,
	}
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
	listings, err := h.pushProductToWoo.Execute(c.Request.Context(), appchannels.PushProductToWooInput{
		ProductID: req.ProductId, ConnectionID: id,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListings(listings))
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

func (h *Handler) GetProductChannelListing(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	listing, err := h.getProductChannelListing.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListing(*listing))
}

func (h *Handler) ListSyncLogs(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	entries, err := h.listSyncLogs.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncLogEntry, len(entries))
	for i, e := range entries {
		resp[i] = toGenSyncLogEntry(e)
	}
	middleware.OK(c, resp)
}

func (h *Handler) ListAllSyncLogs(c *gin.Context, params genintegrations.ListAllSyncLogsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	entries, total, err := h.listAllSyncLogs.Execute(c.Request.Context(), page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncLogEntry, len(entries))
	for i, e := range entries {
		gen := toGenSyncLogEntry(e.SyncLogEntry)
		if e.ProductName != "" {
			gen.ProductName = &e.ProductName
		}
		if e.StoreURL != "" {
			gen.StoreUrl = &e.StoreURL
		}
		resp[i] = gen
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, genintegrations.SyncLogEntryPageResponse{Data: &resp, Total: &totalInt})
}

func (h *Handler) ListSalesChannels(c *gin.Context) {
	list, err := h.listSalesChannels.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SalesChannel, len(list))
	for i, ch := range list {
		resp[i] = toGenSalesChannel(ch)
	}
	middleware.OK(c, resp)
}

func (h *Handler) CreateSalesChannel(c *gin.Context) {
	var req genintegrations.CreateSalesChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	tag := ""
	if req.EnrollmentTag != nil {
		tag = *req.EnrollmentTag
	}
	channel, err := h.createSalesChannel.Execute(c.Request.Context(), appchannels.CreateSalesChannelInput{
		Name: req.Name, Type: channelsDomain.ChannelType(req.Type), ConnectionID: req.ConnectionId,
		EnrollmentMode: channelsDomain.EnrollmentMode(req.EnrollmentMode), EnrollmentCategoryID: req.EnrollmentCategoryId,
		EnrollmentTag: tag,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGenSalesChannel(*channel)})
}

func (h *Handler) UpdateSalesChannel(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpdateSalesChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	tag := ""
	if req.EnrollmentTag != nil {
		tag = *req.EnrollmentTag
	}
	channel, err := h.updateSalesChannel.Execute(c.Request.Context(), appchannels.UpdateSalesChannelInput{
		ID: id, Name: req.Name, ConnectionID: req.ConnectionId,
		EnrollmentMode: channelsDomain.EnrollmentMode(req.EnrollmentMode), EnrollmentCategoryID: req.EnrollmentCategoryId,
		EnrollmentTag: tag, Active: req.Active,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenSalesChannel(*channel))
}

func (h *Handler) DeleteSalesChannel(c *gin.Context, id genintegrations.Id) {
	err := h.deleteSalesChannel.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListSalesChannelProducts(c *gin.Context, id genintegrations.Id, params genintegrations.ListSalesChannelProductsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	rows, total, err := h.listSalesChannelProducts.Execute(c.Request.Context(), appchannels.ListProductsWithChannelEnrollmentInput{
		ChannelID: id, Search: params.Search, Limit: perPage, Offset: (page - 1) * perPage,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SalesChannelProductEnrollment, len(rows))
	for i, r := range rows {
		resp[i] = genintegrations.SalesChannelProductEnrollment{Enrolled: r.Enrolled, Product: toGenSalesChannelProduct(r.Product)}
	}
	totalInt := int(total)
	c.JSON(http.StatusOK, genintegrations.SalesChannelProductListResponse{Data: &resp, Total: &totalInt})
}

func (h *Handler) SetSalesChannelProductOverride(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	var req genintegrations.SetSalesChannelProductOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	_, err := h.setProductChannelOverride.Execute(c.Request.Context(), appchannels.SetProductChannelOverrideInput{
		ChannelID: id, ProductID: productId, Included: req.Included,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
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

func toGenAIConnection(c integrationsDomain.AIConnection) genintegrations.AIConnection {
	resp := genintegrations.AIConnection{
		Id: c.ID, Provider: genintegrations.AIProvider(c.Provider),
		Status: genintegrations.ConnectionStatus(c.Status), CreatedAt: c.CreatedAt,
	}
	if c.BaseURL != "" {
		resp.BaseUrl = &c.BaseURL
	}
	if c.Model != "" {
		resp.Model = &c.Model
	}
	return resp
}

func toGenListing(l channelsDomain.ProductChannelListing) genintegrations.ProductChannelListing {
	isVariation := l.IsVariation
	return genintegrations.ProductChannelListing{
		Id:           l.ID,
		ProductId:    l.ProductID,
		ConnectionId: l.ConnectionID,
		Provider:     l.Provider,
		ExternalId:   l.ExternalID,
		VariantId:    l.VariantID,
		IsVariation:  &isVariation,
		SyncStatus:   genintegrations.ProductChannelListingSyncStatus(l.SyncStatus),
		LastSyncedAt: l.LastSyncedAt,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

func toGenListings(ls []channelsDomain.ProductChannelListing) []genintegrations.ProductChannelListing {
	out := make([]genintegrations.ProductChannelListing, len(ls))
	for i, l := range ls {
		out[i] = toGenListing(l)
	}
	return out
}

func toGenSyncLogEntry(e channelsDomain.SyncLogEntry) genintegrations.SyncLogEntry {
	changes := make(map[string]genintegrations.FieldChange, len(e.Changes))
	for k, v := range e.Changes {
		changes[k] = genintegrations.FieldChange{From: v.From, To: v.To}
	}
	fields := e.Fields
	if fields == nil {
		fields = map[string]string{}
	}
	return genintegrations.SyncLogEntry{
		Id: e.ID, ListingId: e.ListingID, ProductId: e.ProductID, ConnectionId: e.ConnectionID,
		Provider: e.Provider, ExternalId: e.ExternalID,
		Direction: genintegrations.SyncLogEntryDirection(e.Direction), Status: genintegrations.SyncLogEntryStatus(e.Status),
		ErrorMessage: e.ErrorMessage, Fields: fields, Changes: changes, SyncedAt: e.SyncedAt,
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

func toGenSalesChannel(c channelsDomain.SalesChannel) genintegrations.SalesChannel {
	resp := genintegrations.SalesChannel{
		Id: c.ID, Name: c.Name, Type: genintegrations.ChannelType(c.Type),
		ConnectionId: c.ConnectionID, EnrollmentMode: genintegrations.EnrollmentMode(c.EnrollmentMode),
		EnrollmentCategoryId: c.EnrollmentCategoryID,
		Active:               c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.EnrollmentTag != "" {
		resp.EnrollmentTag = &c.EnrollmentTag
	}
	return resp
}

func toGenSalesChannelProduct(p pmsDomain.Product) genintegrations.SalesChannelProduct {
	resp := genintegrations.SalesChannelProduct{Id: p.ID, Name: p.Name, Slug: p.Slug, CategoryId: p.CategoryID}
	if len(p.Tags) > 0 {
		tags := p.Tags
		resp.Tags = &tags
	}
	return resp
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
		errors.Is(err, pmsDomain.ErrProductNotFound),
		errors.Is(err, channelsDomain.ErrListingNotFound),
		errors.Is(err, channelsDomain.ErrSalesChannelNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, integrationsDomain.ErrStoreURLInvalid):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrProviderNotSupported),
		errors.Is(err, integrationsDomain.ErrAPIKeyRequired),
		errors.Is(err, integrationsDomain.ErrBaseURLRequired):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrCannotPushVariableProduct),
		errors.Is(err, channelsDomain.ErrInvalidSyncEntityType),
		errors.Is(err, channelsDomain.ErrInvalidSyncDirection),
		errors.Is(err, channelsDomain.ErrInvalidSyncInterval),
		errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity),
		errors.Is(err, channelsDomain.ErrDefaultWarehouseAndCurrencyRequired),
		errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired),
		errors.Is(err, channelsDomain.ErrInvalidChannelType),
		errors.Is(err, channelsDomain.ErrInvalidEnrollmentMode),
		errors.Is(err, channelsDomain.ErrConnectionRequired),
		errors.Is(err, channelsDomain.ErrEnrollmentCategoryRequired),
		errors.Is(err, channelsDomain.ErrEnrollmentTagRequired):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrStateInvalid):
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, integrationsDomain.ErrConnectionTestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrWooRequestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrNoTenantContext):
		middleware.Fail(c, http.StatusBadRequest, "NO_TENANT", err.Error())
	case errors.Is(err, channelsDomain.ErrImportJobAlreadyActive):
		middleware.Fail(c, http.StatusConflict, "IMPORT_ALREADY_RUNNING", err.Error())
	default:
		log.Printf("integrations: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
