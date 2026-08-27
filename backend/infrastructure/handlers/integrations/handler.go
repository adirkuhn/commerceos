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
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
	bolinfra "github.com/adirkuhn/commerceos/infrastructure/bol"
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
type listWooTaxesUC interface {
	Execute(ctx context.Context, connectionID string) ([]appintegrations.WooTaxClassOption, error)
}
type updateConnectionSettingsUC interface {
	Execute(ctx context.Context, in appintegrations.UpdateConnectionSettingsInput) (*integrationsDomain.Connection, error)
}
type setWebhookSecretUC interface {
	Execute(ctx context.Context, connectionID string) (*appintegrations.SetWebhookSecretResult, error)
}
type listWebhookFallbackEventsUC interface {
	Execute(ctx context.Context, status webhooksDomain.FallbackStatus) ([]webhooksDomain.FallbackEvent, error)
}
type retryWebhookFallbackEventUC interface {
	Execute(ctx context.Context, id string) error
}
type abandonWebhookFallbackEventUC interface {
	Execute(ctx context.Context, id string) error
}
type createWebhookForwardRuleUC interface {
	Execute(ctx context.Context, provider, topic, targetURL string) (*webhooksDomain.ForwardRule, error)
}
type listWebhookForwardRulesUC interface {
	Execute(ctx context.Context) ([]webhooksDomain.ForwardRule, error)
}
type deleteWebhookForwardRuleUC interface {
	Execute(ctx context.Context, id string) error
}
type listWebhookForwardDeliveriesUC interface {
	Execute(ctx context.Context, limit int) ([]webhooksDomain.ForwardDelivery, error)
}
type createImportJobUC interface {
	Execute(ctx context.Context, in appchannels.CreateImportJobInput) (*channelsDomain.ImportJob, error)
}
type pushProductToWooUC interface {
	Execute(ctx context.Context, in appchannels.PushProductToWooInput) ([]channelsDomain.ProductChannelListing, error)
}
type fetchWooOrderUC interface {
	ExecuteOne(ctx context.Context, in appchannels.FetchWooOrderInput) (*appchannels.FetchWooOrderResult, error)
}
type listSyncSchedulesUC interface {
	Execute(ctx context.Context, connectionID string) ([]channelsDomain.SyncSchedule, error)
}
type deleteAllImportedProductsUC interface {
	Execute(ctx context.Context, in appchannels.DeleteAllImportedProductsInput) (*appchannels.DeleteAllImportedProductsResult, error)
}
type upsertSyncSchedulesUC interface {
	Execute(ctx context.Context, in appchannels.UpsertSyncSchedulesInput) ([]channelsDomain.SyncSchedule, error)
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
type listSyncLogsNeedingAttentionUC interface {
	Execute(ctx context.Context, page, perPage int) ([]appchannels.SyncLogEntryWithNames, int64, error)
}
type retryFailedSyncUC interface {
	Execute(ctx context.Context, syncLogEntryID string) error
}
type resyncProductToChannelsUC interface {
	Execute(ctx context.Context, productID string) error
}
type dismissFailedSyncUC interface {
	Execute(ctx context.Context, syncLogEntryID string) error
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
type listWooCategoriesUC interface {
	Execute(ctx context.Context, connectionID string) ([]integrationsDomain.WooCategory, error)
}
type listExternalCategoryLinksUC interface {
	Execute(ctx context.Context, connectionID string) ([]channelsDomain.ExternalCategoryLink, error)
}
type upsertExternalCategoryLinkUC interface {
	Execute(ctx context.Context, in appchannels.UpsertExternalCategoryLinkInput) (*channelsDomain.ExternalCategoryLink, error)
}
type deleteExternalCategoryLinkUC interface {
	Execute(ctx context.Context, id string) error
}
type connectBolUC interface {
	Execute(ctx context.Context, in appintegrations.ConnectBolInput) (*integrationsDomain.Connection, error)
}
type pushProductToBolUC interface {
	Execute(ctx context.Context, in appchannels.PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error)
}
type getBolProductOverrideUC interface {
	Execute(ctx context.Context, productID, connectionID string) (*channelsDomain.BolProductOverride, error)
}
type upsertBolProductOverrideUC interface {
	Execute(ctx context.Context, in appchannels.UpsertBolProductOverrideInput) (*channelsDomain.BolProductOverride, error)
}
type getBolProductContentUC interface {
	Execute(ctx context.Context, productID, connectionID string) (*channelsDomain.BolProductContent, error)
}
type upsertBolProductContentUC interface {
	Execute(ctx context.Context, in appchannels.UpsertBolProductContentInput) (*channelsDomain.BolProductContent, error)
}
type getBolCommissionUC interface {
	Execute(ctx context.Context, in appchannels.GetBolCommissionInput) (*appchannels.BolCommissionResult, error)
}

type listBolEconomicOperatorsUC interface {
	Execute(ctx context.Context, connectionID string) ([]integrationsDomain.BolEconomicOperator, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listConnections              listConnectionsUC
	startWooConnect              startWooConnectUC
	completeWooConnect           completeWooConnectUC
	manualConnectWoo             manualConnectWooCommerceUC
	disconnect                   disconnectConnectionUC
	testConnection               testConnectionUC
	listWooTaxes                 listWooTaxesUC
	createImportJob              createImportJobUC
	pushProductToWoo             pushProductToWooUC
	listSyncSchedules            listSyncSchedulesUC
	upsertSyncSchedules          upsertSyncSchedulesUC
	deleteAllImportedProds       deleteAllImportedProductsUC
	updateConnectionSettings     updateConnectionSettingsUC
	getProductChannelListing     getProductChannelListingUC
	listSyncLogs                 listSyncLogsUC
	listAllSyncLogs              listAllSyncLogsUC
	listSyncLogsNeedingAttention listSyncLogsNeedingAttentionUC
	retryFailedSync              retryFailedSyncUC
	dismissFailedSync            dismissFailedSyncUC
	resyncProductToChannels      resyncProductToChannelsUC
	getAIConnection              getAIConnectionUC
	connectAIProvider            connectAIProviderUC
	disconnectAIConnection       disconnectAIConnectionUC
	listSalesChannels            listSalesChannelsUC
	createSalesChannel           createSalesChannelUC
	updateSalesChannel           updateSalesChannelUC
	deleteSalesChannel           deleteSalesChannelUC
	listSalesChannelProducts     listSalesChannelProductsUC
	setProductChannelOverride    setProductChannelOverrideUC
	fetchWooOrder                fetchWooOrderUC
	listWooCategories            listWooCategoriesUC
	listExternalCategoryLinks    listExternalCategoryLinksUC
	upsertExternalCategoryLink   upsertExternalCategoryLinkUC
	deleteExternalCategoryLink   deleteExternalCategoryLinkUC
	setWebhookSecret             setWebhookSecretUC
	listWebhookFallbackEvents    listWebhookFallbackEventsUC
	retryWebhookFallbackEvent    retryWebhookFallbackEventUC
	abandonWebhookFallbackEvent  abandonWebhookFallbackEventUC
	createWebhookForwardRule     createWebhookForwardRuleUC
	listWebhookForwardRules      listWebhookForwardRulesUC
	deleteWebhookForwardRule     deleteWebhookForwardRuleUC
	listWebhookForwardDeliveries listWebhookForwardDeliveriesUC
	connectBol                   connectBolUC
	pushProductToBol             pushProductToBolUC
	getBolProductOverride        getBolProductOverrideUC
	upsertBolProductOverride     upsertBolProductOverrideUC
	getBolProductContent         getBolProductContentUC
	upsertBolProductContent      upsertBolProductContentUC
	getBolCommission             getBolCommissionUC
	listBolEconomicOperators     listBolEconomicOperatorsUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listConnections listConnectionsUC,
	startWooConnect startWooConnectUC,
	completeWooConnect completeWooConnectUC,
	manualConnectWoo manualConnectWooCommerceUC,
	disconnect disconnectConnectionUC,
	testConnection testConnectionUC,
	listWooTaxes listWooTaxesUC,
	createImportJob createImportJobUC,
	pushProductToWoo pushProductToWooUC,
	listSyncSchedules listSyncSchedulesUC,
	upsertSyncSchedules upsertSyncSchedulesUC,
	deleteAllImportedProds deleteAllImportedProductsUC,
	updateConnectionSettings updateConnectionSettingsUC,
	getProductChannelListing getProductChannelListingUC,
	listSyncLogs listSyncLogsUC,
	listAllSyncLogs listAllSyncLogsUC,
	listSyncLogsNeedingAttention listSyncLogsNeedingAttentionUC,
	retryFailedSync retryFailedSyncUC,
	dismissFailedSync dismissFailedSyncUC,
	resyncProductToChannels resyncProductToChannelsUC,
	getAIConnection getAIConnectionUC,
	connectAIProvider connectAIProviderUC,
	disconnectAIConnection disconnectAIConnectionUC,
	listSalesChannels listSalesChannelsUC,
	createSalesChannel createSalesChannelUC,
	updateSalesChannel updateSalesChannelUC,
	deleteSalesChannel deleteSalesChannelUC,
	listSalesChannelProducts listSalesChannelProductsUC,
	setProductChannelOverride setProductChannelOverrideUC,
	fetchWooOrder fetchWooOrderUC,
	listWooCategories listWooCategoriesUC,
	listExternalCategoryLinks listExternalCategoryLinksUC,
	upsertExternalCategoryLink upsertExternalCategoryLinkUC,
	deleteExternalCategoryLink deleteExternalCategoryLinkUC,
	setWebhookSecret setWebhookSecretUC,
	listWebhookFallbackEvents listWebhookFallbackEventsUC,
	retryWebhookFallbackEvent retryWebhookFallbackEventUC,
	abandonWebhookFallbackEvent abandonWebhookFallbackEventUC,
	createWebhookForwardRule createWebhookForwardRuleUC,
	listWebhookForwardRules listWebhookForwardRulesUC,
	deleteWebhookForwardRule deleteWebhookForwardRuleUC,
	listWebhookForwardDeliveries listWebhookForwardDeliveriesUC,
	connectBol connectBolUC,
	pushProductToBol pushProductToBolUC,
	getBolProductOverride getBolProductOverrideUC,
	upsertBolProductOverride upsertBolProductOverrideUC,
	getBolProductContent getBolProductContentUC,
	upsertBolProductContent upsertBolProductContentUC,
	getBolCommission getBolCommissionUC,
	listBolEconomicOperators listBolEconomicOperatorsUC,
) *Handler {
	return &Handler{
		listConnections:              listConnections,
		startWooConnect:              startWooConnect,
		completeWooConnect:           completeWooConnect,
		manualConnectWoo:             manualConnectWoo,
		disconnect:                   disconnect,
		testConnection:               testConnection,
		listWooTaxes:                 listWooTaxes,
		createImportJob:              createImportJob,
		pushProductToWoo:             pushProductToWoo,
		listSyncSchedules:            listSyncSchedules,
		upsertSyncSchedules:          upsertSyncSchedules,
		deleteAllImportedProds:       deleteAllImportedProds,
		updateConnectionSettings:     updateConnectionSettings,
		getProductChannelListing:     getProductChannelListing,
		listSyncLogs:                 listSyncLogs,
		listAllSyncLogs:              listAllSyncLogs,
		listSyncLogsNeedingAttention: listSyncLogsNeedingAttention,
		retryFailedSync:              retryFailedSync,
		dismissFailedSync:            dismissFailedSync,
		resyncProductToChannels:      resyncProductToChannels,
		getAIConnection:              getAIConnection,
		connectAIProvider:            connectAIProvider,
		disconnectAIConnection:       disconnectAIConnection,
		listSalesChannels:            listSalesChannels,
		createSalesChannel:           createSalesChannel,
		updateSalesChannel:           updateSalesChannel,
		deleteSalesChannel:           deleteSalesChannel,
		listSalesChannelProducts:     listSalesChannelProducts,
		setProductChannelOverride:    setProductChannelOverride,
		fetchWooOrder:                fetchWooOrder,
		listWooCategories:            listWooCategories,
		listExternalCategoryLinks:    listExternalCategoryLinks,
		upsertExternalCategoryLink:   upsertExternalCategoryLink,
		deleteExternalCategoryLink:   deleteExternalCategoryLink,
		setWebhookSecret:             setWebhookSecret,
		listWebhookFallbackEvents:    listWebhookFallbackEvents,
		retryWebhookFallbackEvent:    retryWebhookFallbackEvent,
		abandonWebhookFallbackEvent:  abandonWebhookFallbackEvent,
		createWebhookForwardRule:     createWebhookForwardRule,
		listWebhookForwardRules:      listWebhookForwardRules,
		deleteWebhookForwardRule:     deleteWebhookForwardRule,
		listWebhookForwardDeliveries: listWebhookForwardDeliveries,
		connectBol:                   connectBol,
		pushProductToBol:             pushProductToBol,
		getBolProductOverride:        getBolProductOverride,
		upsertBolProductOverride:     upsertBolProductOverride,
		getBolProductContent:         getBolProductContent,
		upsertBolProductContent:      upsertBolProductContent,
		getBolCommission:             getBolCommission,
		listBolEconomicOperators:     listBolEconomicOperators,
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
	webhookPublisher appwebhooks.Publisher,
) {
	connRepo := repositories.NewConnectionRepo(db)
	webhookFallbackRepo := repositories.NewWebhookFallbackRepo(db)
	webhookForwardRuleRepo := repositories.NewWebhookForwardRuleRepo(db)
	webhookForwardDeliveryRepo := repositories.NewWebhookForwardDeliveryRepo(db)
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
	mediaRepo := repositories.NewMediaRepo(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	salesOrderLineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	salesOrderLinkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	stockRecorderUC := appinventory.NewRecordStockMovementUseCase(db, stockMovementRepo, outboxRepo)
	allocateStockUC := appinventory.NewAllocateSaleStockUseCase(db, stockMovementRepo, batchRepo, outboxRepo)
	deleteSalesOrderUC := appoms.NewDeleteSalesOrderUseCase(
		db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, stockRecorderUC,
	)
	customerRepo := repositories.NewCustomerRepo(db)
	fetchWooOrderUC := appchannels.NewImportWooOrdersUseCase(
		db, connRepo, wooFactory, encryptionSecret,
		salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, customerRepo,
		allocateStockUC, appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, allocateStockUC, stockRecorderUC),
	)

	stockLevelUC := appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, repositories.NewBatchRepo(db), repositories.NewTenantSettingsRepo(db))
	categoryLinkRepo := repositories.NewExternalCategoryLinkRepo(db)
	resolveCategoryLinkUC := appchannels.NewResolveCategoryLinkUseCase(db, categoryLinkRepo, categoryRepo)
	pushProductToWooUC := appchannels.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, encryptionSecret,
		productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo,
		stockLevelUC,
		mediaRepo,
		resolveCategoryLinkUC,
	)
	bolOverrideRepo := repositories.NewBolProductOverrideRepo(db)
	bolContentRepo := repositories.NewBolProductContentRepo(db)
	pushProductToBolUC := appchannels.NewPushProductToBolUseCase(
		db, connRepo, bolinfra.NewClientFactory(), encryptionSecret, productRepo, variantRepo, skuRepo, listingRepo, bolOverrideRepo, bolContentRepo, syncLogRepo, stockLevelUC,
		appnotifications.NewCreateNotificationUseCase(db, notificationRepo),
	)
	syncProductToChannelsUC := appchannels.NewSyncProductToChannelsUseCase(
		db, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, connRepo, wooFactory, encryptionSecret, pushProductToWooUC, mediaRepo, resolveCategoryLinkUC, pushProductToBolUC,
	)
	syncStockToChannelsUC := appchannels.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, syncLogRepo, connRepo, wooFactory, stockLevelUC, encryptionSecret, pushProductToBolUC,
	)
	enrollProductInChannelsUC := appchannels.NewEnrollProductInChannelsUseCase(
		db, productRepo, salesChannelRepo, salesChannelOverrideRepo, pushProductToWooUC, pushProductToBolUC,
	)
	resyncProductToChannelsUC := appchannels.NewResyncProductToChannelsUseCase(
		db, variantRepo, skuRepo, syncProductToChannelsUC, syncStockToChannelsUC, enrollProductInChannelsUC,
	)

	h := NewHandler(
		appintegrations.NewListConnectionsUseCase(db, connRepo),
		appintegrations.NewStartWooConnectUseCase(stateSecret, publicAPIBaseURL, frontendBaseURL),
		appintegrations.NewCompleteWooConnectUseCase(db, connRepo, outboxRepo, stateSecret, encryptionSecret),
		appintegrations.NewManualConnectWooCommerceUseCase(db, connRepo, outboxRepo, encryptionSecret),
		appintegrations.NewDisconnectConnectionUseCase(db, connRepo),
		appintegrations.NewTestConnectionUseCase(db, connRepo, wooFactory, encryptionSecret),
		appintegrations.NewListWooTaxesUseCase(db, connRepo, wooFactory, encryptionSecret),
		appchannels.NewCreateImportJobUseCase(
			db, importJobRepo, listingRepo, appnotifications.NewCreateNotificationUseCase(db, notificationRepo),
		),
		pushProductToWooUC,
		appchannels.NewListSyncSchedulesUseCase(db, scheduleRepo),
		appchannels.NewUpsertSyncSchedulesUseCase(db, scheduleRepo),
		appchannels.NewDeleteAllImportedProductsUseCase(db, listingRepo, productRepo, categoryRepo, salesOrderRepo, deleteSalesOrderUC),
		appintegrations.NewUpdateConnectionSettingsUseCase(db, connRepo),
		appchannels.NewGetProductChannelListingUseCase(db, listingRepo),
		appchannels.NewListSyncLogsUseCase(db, listingRepo, syncLogRepo),
		appchannels.NewListAllSyncLogsUseCase(db, syncLogRepo, productRepo, connRepo),
		appchannels.NewListSyncLogsNeedingAttentionUseCase(db, syncLogRepo, productRepo, connRepo),
		appchannels.NewRetryFailedSyncUseCase(db, syncLogRepo, pushProductToWooUC),
		appchannels.NewDismissFailedSyncUseCase(db, syncLogRepo),
		resyncProductToChannelsUC,
		appintegrations.NewGetAIConnectionUseCase(db, aiConnRepo),
		appintegrations.NewConnectAIProviderUseCase(db, aiConnRepo, encryptionSecret),
		appintegrations.NewDisconnectAIConnectionUseCase(db, aiConnRepo),
		appchannels.NewListSalesChannelsUseCase(db, salesChannelRepo),
		appchannels.NewCreateSalesChannelUseCase(db, salesChannelRepo, connRepo),
		appchannels.NewUpdateSalesChannelUseCase(db, salesChannelRepo, connRepo),
		appchannels.NewDeleteSalesChannelUseCase(db, salesChannelRepo),
		appchannels.NewListProductsWithChannelEnrollmentUseCase(db, productRepo, salesChannelRepo, salesChannelOverrideRepo),
		appchannels.NewSetProductChannelOverrideUseCase(db, salesChannelOverrideRepo, salesChannelRepo, productRepo),
		fetchWooOrderUC,
		appintegrations.NewListWooCategoriesUseCase(db, connRepo, wooFactory, encryptionSecret),
		appchannels.NewListExternalCategoryLinksUseCase(db, categoryLinkRepo),
		appchannels.NewUpsertExternalCategoryLinkUseCase(db, categoryLinkRepo, categoryRepo),
		appchannels.NewDeleteExternalCategoryLinkUseCase(db, categoryLinkRepo),
		appintegrations.NewSetWebhookSecretUseCase(db, connRepo, encryptionSecret),
		appwebhooks.NewListWebhookFallbackEventsUseCase(db, webhookFallbackRepo),
		appwebhooks.NewRetryWebhookFallbackEventUseCase(db, webhookFallbackRepo, webhookPublisher),
		appwebhooks.NewAbandonWebhookFallbackEventUseCase(db, webhookFallbackRepo),
		appwebhooks.NewCreateForwardRuleUseCase(db, webhookForwardRuleRepo),
		appwebhooks.NewListForwardRulesUseCase(db, webhookForwardRuleRepo),
		appwebhooks.NewDeleteForwardRuleUseCase(db, webhookForwardRuleRepo),
		appwebhooks.NewListForwardDeliveriesUseCase(db, webhookForwardDeliveryRepo),
		appintegrations.NewConnectBolUseCase(db, connRepo, outboxRepo, encryptionSecret),
		pushProductToBolUC,
		appchannels.NewGetBolProductOverrideUseCase(db, bolOverrideRepo),
		appchannels.NewUpsertBolProductOverrideUseCase(db, bolOverrideRepo, outboxRepo),
		appchannels.NewGetBolProductContentUseCase(db, bolContentRepo),
		appchannels.NewUpsertBolProductContentUseCase(db, bolContentRepo, outboxRepo),
		appchannels.NewGetBolCommissionUseCase(db, connRepo, bolinfra.NewClientFactory(), encryptionSecret, skuRepo, bolOverrideRepo),
		appintegrations.NewListBolEconomicOperatorsUseCase(db, connRepo, bolinfra.NewClientFactory(), encryptionSecret),
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

func (h *Handler) ListWooTaxes(c *gin.Context, id genintegrations.Id) {
	options, err := h.listWooTaxes.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.WooTaxClass, len(options))
	for i, o := range options {
		resp[i] = genintegrations.WooTaxClass{Slug: o.Slug, Label: o.Label}
	}
	middleware.OK(c, resp)
}

func (h *Handler) ListWooCategories(c *gin.Context, id genintegrations.Id) {
	categories, err := h.listWooCategories.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.WooCategory, len(categories))
	for i, cat := range categories {
		resp[i] = genintegrations.WooCategory{ExternalId: cat.ExternalID, Name: cat.Name, Slug: cat.Slug}
	}
	middleware.OK(c, resp)
}

func (h *Handler) ListExternalCategoryLinks(c *gin.Context, id genintegrations.Id) {
	links, err := h.listExternalCategoryLinks.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.ExternalCategoryLink, len(links))
	for i, l := range links {
		resp[i] = toGenExternalCategoryLink(l)
	}
	middleware.OK(c, resp)
}

func (h *Handler) UpsertExternalCategoryLink(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpsertExternalCategoryLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	link, err := h.upsertExternalCategoryLink.Execute(c.Request.Context(), appchannels.UpsertExternalCategoryLinkInput{
		ConnectionID: id, ExternalID: req.ExternalId, CategoryID: req.CategoryId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenExternalCategoryLink(*link))
}

func (h *Handler) DeleteExternalCategoryLink(c *gin.Context, id genintegrations.Id, linkId genintegrations.LinkId) {
	err := h.deleteExternalCategoryLink.Execute(c.Request.Context(), linkId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenExternalCategoryLink(l channelsDomain.ExternalCategoryLink) genintegrations.ExternalCategoryLink {
	return genintegrations.ExternalCategoryLink{
		Id: l.ID, ConnectionId: l.ConnectionID, Provider: l.Provider,
		ExternalId: l.ExternalID, CategoryId: l.CategoryID,
		CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
}

func (h *Handler) UpdateConnectionSettings(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpdateConnectionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	conn, err := h.updateConnectionSettings.Execute(c.Request.Context(), appintegrations.UpdateConnectionSettingsInput{
		ID: id, DefaultCurrency: req.DefaultCurrency, DefaultWarehouseID: req.DefaultWarehouseId,
		BolEconomicOperatorID: req.BolEconomicOperatorId, BolDeliveryCode: req.BolDeliveryCode,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) SetWebhookSecret(c *gin.Context, id genintegrations.Id) {
	result, err := h.setWebhookSecret.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"connection_id": result.ConnectionID, "webhook_secret": result.PlaintextSecret})
}

func (h *Handler) ListWebhookFallbackEvents(c *gin.Context, params genintegrations.ListWebhookFallbackEventsParams) {
	status := webhooksDomain.FallbackStatusPendingRepublish
	if params.Status != nil {
		status = webhooksDomain.FallbackStatus(*params.Status)
	}
	events, err := h.listWebhookFallbackEvents.Execute(c.Request.Context(), status)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookFallbackEvent, len(events))
	for i, e := range events {
		id, eventID, provider := e.ID, e.EventID, e.Provider
		st := genintegrations.WebhookFallbackEventStatus(e.Status)
		receivedAt := e.ReceivedAt
		out[i] = genintegrations.WebhookFallbackEvent{
			Id: &id, EventId: &eventID, Provider: &provider, Status: &st, ReceivedAt: &receivedAt,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) ListWebhookForwardRules(c *gin.Context) {
	rules, err := h.listWebhookForwardRules.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookForwardRule, len(rules))
	for i, r := range rules {
		id, provider, topic, targetURL, active, createdAt := r.ID, r.Provider, r.Topic, r.TargetURL, r.Active, r.CreatedAt
		out[i] = genintegrations.WebhookForwardRule{
			Id: &id, Provider: &provider, Topic: &topic, TargetUrl: &targetURL, Active: &active, CreatedAt: &createdAt,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) CreateWebhookForwardRule(c *gin.Context) {
	var req genintegrations.CreateWebhookForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	topic := ""
	if req.Topic != nil {
		topic = *req.Topic
	}
	rule, err := h.createWebhookForwardRule.Execute(c.Request.Context(), req.Provider, topic, req.TargetUrl)
	if err != nil {
		if errors.Is(err, webhooksDomain.ErrInvalidTargetURL) || errors.Is(err, webhooksDomain.ErrProviderRequired) {
			middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
			return
		}
		if err := mapError(c, err); err != nil {
			return
		}
	}
	middleware.OK(c, genintegrations.WebhookForwardRule{
		Id: &rule.ID, Provider: &rule.Provider, Topic: &rule.Topic, TargetUrl: &rule.TargetURL,
		Active: &rule.Active, CreatedAt: &rule.CreatedAt,
	})
}

func (h *Handler) DeleteWebhookForwardRule(c *gin.Context, id genintegrations.Id) {
	err := h.deleteWebhookForwardRule.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListWebhookForwardDeliveries(c *gin.Context, params genintegrations.ListWebhookForwardDeliveriesParams) {
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}
	deliveries, err := h.listWebhookForwardDeliveries.Execute(c.Request.Context(), limit)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genintegrations.WebhookForwardDelivery, len(deliveries))
	for i, d := range deliveries {
		id, ruleID, provider, topic, targetURL := d.ID, d.RuleID, d.Provider, d.Topic, d.TargetURL
		status := genintegrations.WebhookForwardDeliveryStatus(d.Status)
		attempts, lastError, nextAttemptAt, createdAt := d.Attempts, d.LastError, d.NextAttemptAt, d.CreatedAt
		out[i] = genintegrations.WebhookForwardDelivery{
			Id: &id, RuleId: &ruleID, Provider: &provider, Topic: &topic, TargetUrl: &targetURL,
			Status: &status, Attempts: &attempts, LastError: &lastError,
			NextAttemptAt: &nextAttemptAt, DeliveredAt: d.DeliveredAt, CreatedAt: &createdAt,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) RetryWebhookFallbackEvent(c *gin.Context, id genintegrations.Id) {
	err := h.retryWebhookFallbackEvent.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, webhooksDomain.ErrFallbackEventNotFound) {
			middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		middleware.Fail(c, http.StatusBadGateway, "PUBLISH_FAILED", err.Error())
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) AbandonWebhookFallbackEvent(c *gin.Context, id genintegrations.Id) {
	err := h.abandonWebhookFallbackEvent.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
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

func (h *Handler) FetchWooOrder(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.FetchWooOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.ExternalOrderId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "external_order_id is required")
		return
	}
	result, err := h.fetchWooOrder.ExecuteOne(c.Request.Context(), appchannels.FetchWooOrderInput{
		ConnectionID: id, ExternalOrderID: req.ExternalOrderId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	var salesOrderID *string
	if result.SalesOrderID != "" {
		salesOrderID = &result.SalesOrderID
	}
	middleware.OK(c, genintegrations.FetchWooOrderResult{
		ExternalOrderId: result.ExternalOrderID, SalesOrderId: salesOrderID,
		Created: result.Created, Updated: result.Updated, Unchanged: result.Unchanged,
		Errors: emptySliceIfNil(result.Errors),
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
	listings, err := h.pushProductToWoo.Execute(c.Request.Context(), appchannels.PushProductToWooInput{
		ProductID: req.ProductId, ConnectionID: id,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListings(listings))
}

func (h *Handler) ConnectBol(c *gin.Context) {
	var req genintegrations.ConnectBolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.ClientId == "" || req.ClientSecret == "" || req.EconomicOperatorId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "client_id, client_secret, and economic_operator_id are required")
		return
	}
	conn, err := h.connectBol.Execute(c.Request.Context(), appintegrations.ConnectBolInput{
		ClientID: req.ClientId, ClientSecret: req.ClientSecret, EconomicOperatorID: req.EconomicOperatorId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenConnection(*conn))
}

func (h *Handler) PushProductToBol(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.PushProductToWooRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.ProductId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "product_id is required")
		return
	}
	listings, err := h.pushProductToBol.Execute(c.Request.Context(), appchannels.PushProductToBolInput{
		ProductID: req.ProductId, ConnectionID: id,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListings(listings))
}

func (h *Handler) GetBolProductContent(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	content, err := h.getBolProductContent.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	if content == nil {
		middleware.OK(c, nil)
		return
	}
	middleware.OK(c, toGenBolProductContent(*content))
}

func (h *Handler) UpsertBolProductContent(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	var req genintegrations.UpsertBolProductContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	content, err := h.upsertBolProductContent.Execute(c.Request.Context(), appchannels.UpsertBolProductContentInput{
		ProductID: productId, ConnectionID: id,
		ManufacturerContact: req.ManufacturerContact, ManufacturerAddress: req.ManufacturerAddress, ManufacturerName: req.ManufacturerName,
		ShelfLifeClosed: req.ShelfLifeClosed, ShelfLifeAfterOpening: req.ShelfLifeAfterOpening, Ingredients: req.Ingredients,
		ContentAmount: req.ContentAmount, ContentUnit: req.ContentUnit, CountryOfOrigin: req.CountryOfOrigin,
		Substance: req.Substance, InstructionLanguage: req.InstructionLanguage, PackagingLanguage: req.PackagingLanguage,
		ItemsPerPackage: intToInt32Ptr(req.ItemsPerPackage), PackageInsertURL: req.PackageInsertUrl, IsGift: req.IsGift,
		DermatologicallyTested: req.DermatologicallyTested, HazardStatements: req.HazardStatements, SkinType: req.SkinType,
		Hypoallergenic: req.Hypoallergenic, ContainsZincOxide: req.ContainsZincOxide,
		ContentPerDispenserAmount: req.ContentPerDispenserAmount, ContentPerDispenserUnit: req.ContentPerDispenserUnit,
		Brand: req.Brand, FragranceFree: req.FragranceFree, PHNeutral: req.PhNeutral, CoolingEffect: req.CoolingEffect,
		PackageWidthMM: intToInt32Ptr(req.PackageWidthMm), PackageHeightMM: intToInt32Ptr(req.PackageHeightMm),
		PackageLengthMM: intToInt32Ptr(req.PackageLengthMm), PackageWeightG: intToInt32Ptr(req.PackageWeightG),
		PrecautionStatements: req.PrecautionStatements, Precautions: req.Precautions, Warnings: req.Warnings, VideoURL: req.VideoUrl,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenBolProductContent(*content))
}

func (h *Handler) GetBolProductOverride(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	override, err := h.getBolProductOverride.Execute(c.Request.Context(), productId, id)
	if err := mapError(c, err); err != nil {
		return
	}
	if override == nil {
		middleware.OK(c, nil)
		return
	}
	middleware.OK(c, toGenBolOverride(*override))
}

func (h *Handler) UpsertBolProductOverride(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId) {
	var req genintegrations.UpsertBolProductOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	condition, fulfilment := "", ""
	if req.ConditionCategory != nil {
		condition = string(*req.ConditionCategory)
	}
	if req.FulfilmentMethod != nil {
		fulfilment = string(*req.FulfilmentMethod)
	}
	var basisPoints *int32
	if req.CommissionOverrideBasisPoints != nil {
		bp := int32(*req.CommissionOverrideBasisPoints)
		basisPoints = &bp
	}
	override, err := h.upsertBolProductOverride.Execute(c.Request.Context(), appchannels.UpsertBolProductOverrideInput{
		ProductID: productId, ConnectionID: id,
		EANOverride: req.EanOverride, GPCCode: req.GpcCode,
		ConditionCategory:             condition,
		FulfilmentMethod:              fulfilment,
		CommissionOverrideBasisPoints: basisPoints,
		ReferenceOverride:             req.ReferenceOverride,
		CustomPriceMinorUnits:         req.CustomPriceMinorUnits,
		NameOverride:                  req.NameOverride,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenBolOverride(*override))
}

func (h *Handler) GetBolCommission(c *gin.Context, id genintegrations.Id, productId genintegrations.ProductId, skuId string) {
	result, err := h.getBolCommission.Execute(c.Request.Context(), appchannels.GetBolCommissionInput{
		ProductID: productId, ConnectionID: id, SKUID: skuId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genintegrations.BolCommission{
		Ean: result.EAN, FixedAmount: result.FixedAmount, Percentage: result.Percentage, Overridden: result.Overridden,
	})
}

func (h *Handler) ListBolEconomicOperators(c *gin.Context, id genintegrations.Id) {
	operators, err := h.listBolEconomicOperators.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.BolEconomicOperator, len(operators))
	for i, o := range operators {
		resp[i] = genintegrations.BolEconomicOperator{
			Id: o.ID, Name: o.Name, Status: genintegrations.BolEconomicOperatorStatus(o.Status),
		}
	}
	middleware.OK(c, resp)
}

func (h *Handler) DeleteAllImportedProducts(c *gin.Context, id genintegrations.Id) {
	result, err := h.deleteAllImportedProds.Execute(c.Request.Context(), appchannels.DeleteAllImportedProductsInput{ConnectionID: id})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genintegrations.DeleteAllImportedProductsResult{
		DeletedSalesOrders: result.DeletedSalesOrders,
		DeletedProducts:    result.DeletedProducts,
		DeletedCategories:  result.DeletedCategories,
		Errors:             emptySliceIfNil(result.Errors),
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

// UpsertSyncSchedules saves the whole sync-schedule grid in one
// all-or-nothing batch (see UpsertSyncSchedulesUseCase's doc comment) --
// either every row in the request saves, or none of them do, so a failing
// row can never silently block a different row's change from being
// attempted at all.
func (h *Handler) UpsertSyncSchedules(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpsertSyncSchedulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	inputs := make([]appchannels.UpsertSyncScheduleInput, len(req.Schedules))
	for i, s := range req.Schedules {
		inputs[i] = appchannels.UpsertSyncScheduleInput{
			ConnectionID:       id,
			EntityType:         channelsDomain.SyncEntityType(s.EntityType),
			Direction:          channelsDomain.SyncDirection(s.Direction),
			Enabled:            s.Enabled,
			IntervalMinutes:    s.IntervalMinutes,
			DefaultWarehouseID: s.DefaultWarehouseId,
			DefaultCurrency:    s.DefaultCurrency,
		}
	}
	updated, err := h.upsertSyncSchedules.Execute(c.Request.Context(), appchannels.UpsertSyncSchedulesInput{Schedules: inputs})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.SyncSchedule, len(updated))
	for i, s := range updated {
		resp[i] = toGenSyncSchedule(s)
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

func (h *Handler) ListSyncLogsNeedingAttention(c *gin.Context, params genintegrations.ListSyncLogsNeedingAttentionParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	entries, total, err := h.listSyncLogsNeedingAttention.Execute(c.Request.Context(), page, perPage)
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

func (h *Handler) RetryFailedSync(c *gin.Context, id genintegrations.Id) {
	if err := h.retryFailedSync.Execute(c.Request.Context(), id); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genintegrations.EmptyResponse{})
}

func (h *Handler) DismissFailedSync(c *gin.Context, id genintegrations.Id) {
	if err := h.dismissFailedSync.Execute(c.Request.Context(), id); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genintegrations.EmptyResponse{})
}

func (h *Handler) ResyncProductToChannels(c *gin.Context, productId genintegrations.ProductId) {
	if err := h.resyncProductToChannels.Execute(c.Request.Context(), productId); err != nil {
		mapError(c, err)
		return
	}
	c.JSON(http.StatusOK, genintegrations.EmptyResponse{})
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
// WooCallback is the single unauthenticated, server-to-server route Woo
// calls after a merchant approves the connect flow -- there's no gin
// access-log middleware mounted anywhere in this app (see cmd/api/main.go),
// and mapError only logs truly unhandled errors, so a known failure here
// (e.g. a state-signature mismatch from JWT_SECRET changing between the
// connect attempt starting and this callback landing) would otherwise
// leave zero trace of ever having been hit. Always log the outcome --
// low-volume, security-sensitive endpoint, worth the audit trail
// unconditionally. Never logs req.ConsumerKey/ConsumerSecret themselves.
func (h *Handler) WooCallback(c *gin.Context) {
	var req gencallback.WooCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("wooCallback: bind request: %v", err)
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	conn, err := h.completeWooConnect.Execute(c.Request.Context(), appintegrations.CompleteWooConnectInput{
		State:          req.UserId,
		ConsumerKey:    req.ConsumerKey,
		ConsumerSecret: req.ConsumerSecret,
	})
	if err != nil {
		log.Printf("wooCallback: %v", err)
		mapError(c, err)
		return
	}
	log.Printf("wooCallback: connected connection_id=%s", conn.ID)
	middleware.OK(c, gin.H{"connection_id": conn.ID})
}

// ── Domain → generated-type mapper ───────────────────────────────────────────

func toGenConnection(c integrationsDomain.Connection) genintegrations.Connection {
	currency := c.DefaultCurrency
	return genintegrations.Connection{
		Id:                    c.ID,
		Provider:              string(c.Provider),
		StoreUrl:              c.StoreURL.String(),
		Status:                genintegrations.ConnectionStatus(c.Status),
		DefaultCurrency:       &currency,
		DefaultWarehouseId:    c.DefaultWarehouseID,
		BolEconomicOperatorId: c.BolEconomicOperatorID,
		BolDeliveryCode:       c.BolDeliveryCode,
		CreatedAt:             c.CreatedAt,
	}
}

func int32ToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

func intToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := int32(*v)
	return &i
}

func toGenBolProductContent(c channelsDomain.BolProductContent) genintegrations.BolProductContent {
	return genintegrations.BolProductContent{
		ProductId: c.ProductID, ConnectionId: c.ConnectionID,
		ManufacturerContact: c.ManufacturerContact, ManufacturerAddress: c.ManufacturerAddress, ManufacturerName: c.ManufacturerName,
		ShelfLifeClosed: c.ShelfLifeClosed, ShelfLifeAfterOpening: c.ShelfLifeAfterOpening, Ingredients: c.Ingredients,
		ContentAmount: c.ContentAmount, ContentUnit: c.ContentUnit, CountryOfOrigin: c.CountryOfOrigin,
		Substance: c.Substance, InstructionLanguage: c.InstructionLanguage, PackagingLanguage: c.PackagingLanguage,
		ItemsPerPackage: int32ToIntPtr(c.ItemsPerPackage), PackageInsertUrl: c.PackageInsertURL, IsGift: c.IsGift,
		DermatologicallyTested: c.DermatologicallyTested, HazardStatements: c.HazardStatements, SkinType: c.SkinType,
		Hypoallergenic: c.Hypoallergenic, ContainsZincOxide: c.ContainsZincOxide,
		ContentPerDispenserAmount: c.ContentPerDispenserAmount, ContentPerDispenserUnit: c.ContentPerDispenserUnit,
		Brand: c.Brand, FragranceFree: c.FragranceFree, PhNeutral: c.PHNeutral, CoolingEffect: c.CoolingEffect,
		PackageWidthMm: int32ToIntPtr(c.PackageWidthMM), PackageHeightMm: int32ToIntPtr(c.PackageHeightMM),
		PackageLengthMm: int32ToIntPtr(c.PackageLengthMM), PackageWeightG: int32ToIntPtr(c.PackageWeightG),
		PrecautionStatements: c.PrecautionStatements, Precautions: c.Precautions, Warnings: c.Warnings, VideoUrl: c.VideoURL,
	}
}

func toGenBolOverride(o channelsDomain.BolProductOverride) genintegrations.BolProductOverride {
	resp := genintegrations.BolProductOverride{
		ProductId: o.ProductID, ConnectionId: o.ConnectionID,
		EanOverride: o.EANOverride, GpcCode: o.GPCCode,
		ConditionCategory:     genintegrations.BolProductOverrideConditionCategory(o.ConditionCategory),
		FulfilmentMethod:      genintegrations.BolProductOverrideFulfilmentMethod(o.FulfilmentMethod),
		ReferenceOverride:     o.ReferenceOverride,
		CustomPriceMinorUnits: o.CustomPriceMinorUnits,
		NameOverride:          o.NameOverride,
	}
	if o.CommissionOverrideBasisPoints != nil {
		bp := int(*o.CommissionOverrideBasisPoints)
		resp.CommissionOverrideBasisPoints = &bp
	}
	return resp
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
	note := l.NotForSaleNote
	return genintegrations.ProductChannelListing{
		Id:             l.ID,
		ProductId:      l.ProductID,
		ConnectionId:   l.ConnectionID,
		Provider:       l.Provider,
		ExternalId:     l.ExternalID,
		VariantId:      l.VariantID,
		IsVariation:    &isVariation,
		SyncStatus:     genintegrations.ProductChannelListingSyncStatus(l.SyncStatus),
		LastSyncedAt:   l.LastSyncedAt,
		NotForSaleNote: &note,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
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
		ErrorMessage: e.ErrorMessage, Fields: fields, Changes: changes, SyncedAt: e.SyncedAt, DismissedAt: e.DismissedAt,
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
		errors.Is(err, pmsDomain.ErrSKUNotFound),
		errors.Is(err, pmsDomain.ErrCategoryNotFound),
		errors.Is(err, channelsDomain.ErrListingNotFound),
		errors.Is(err, channelsDomain.ErrSalesChannelNotFound),
		errors.Is(err, channelsDomain.ErrExternalCategoryLinkNotFound),
		errors.Is(err, channelsDomain.ErrSyncLogNotFound),
		errors.Is(err, webhooksDomain.ErrFallbackEventNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, channelsDomain.ErrSyncLogHasNoListing):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrStoreURLInvalid):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, integrationsDomain.ErrProviderNotSupported),
		errors.Is(err, integrationsDomain.ErrAPIKeyRequired),
		errors.Is(err, integrationsDomain.ErrBaseURLRequired),
		errors.Is(err, integrationsDomain.ErrInvalidBolDeliveryCode):
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
		errors.Is(err, channelsDomain.ErrEnrollmentTagRequired),
		errors.Is(err, channelsDomain.ErrChannelTypeConnectionMismatch):
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
	case errors.Is(err, channelsDomain.ErrBolMissingEAN),
		errors.Is(err, channelsDomain.ErrCannotPushBulkPoolSKU):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, channelsDomain.ErrBolCommissionRequestFailed),
		errors.Is(err, integrationsDomain.ErrBolEconomicOperatorsRequestFailed):
		middleware.Fail(c, http.StatusBadGateway, "UPSTREAM_ERROR", err.Error())
	default:
		log.Printf("integrations: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
