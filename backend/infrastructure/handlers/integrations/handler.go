package integrations

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	gencallback "github.com/adirkuhn/commerceos/api/integrationscallback"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppayments "github.com/adirkuhn/commerceos/application/payments"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	bolinfra "github.com/adirkuhn/commerceos/infrastructure/bol"
	"github.com/adirkuhn/commerceos/infrastructure/evolutionapi"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listConnections                 listConnectionsUC
	startWooConnect                 startWooConnectUC
	completeWooConnect              completeWooConnectUC
	manualConnectWoo                manualConnectWooCommerceUC
	disconnect                      disconnectConnectionUC
	testConnection                  testConnectionUC
	listWooTaxes                    listWooTaxesUC
	createImportJob                 createImportJobUC
	pushProductToWoo                pushProductToWooUC
	listSyncSchedules               listSyncSchedulesUC
	upsertSyncSchedules             upsertSyncSchedulesUC
	deleteAllImportedProds          deleteAllImportedProductsUC
	updateConnectionSettings        updateConnectionSettingsUC
	getProductChannelListing        getProductChannelListingUC
	listProductChannelListings      listProductChannelListingsUC
	listSyncLogs                    listSyncLogsUC
	listAllSyncLogs                 listAllSyncLogsUC
	listSyncLogsNeedingAttention    listSyncLogsNeedingAttentionUC
	retryFailedSync                 retryFailedSyncUC
	dismissFailedSync               dismissFailedSyncUC
	resyncProductToChannels         resyncProductToChannelsUC
	getAIConnection                 getAIConnectionUC
	connectAIProvider               connectAIProviderUC
	disconnectAIConnection          disconnectAIConnectionUC
	listSalesChannels               listSalesChannelsUC
	createSalesChannel              createSalesChannelUC
	updateSalesChannel              updateSalesChannelUC
	setSalesChannelPaymentMethod    setSalesChannelPaymentMethodUC
	deleteSalesChannel              deleteSalesChannelUC
	listSalesChannelProducts        listSalesChannelProductsUC
	setProductChannelOverride       setProductChannelOverrideUC
	fetchWooOrder                   fetchWooOrderUC
	listWooCategories               listWooCategoriesUC
	listExternalCategoryLinks       listExternalCategoryLinksUC
	upsertExternalCategoryLink      upsertExternalCategoryLinkUC
	deleteExternalCategoryLink      deleteExternalCategoryLinkUC
	setWebhookSecret                setWebhookSecretUC
	listWebhookFallbackEvents       listWebhookFallbackEventsUC
	retryWebhookFallbackEvent       retryWebhookFallbackEventUC
	abandonWebhookFallbackEvent     abandonWebhookFallbackEventUC
	createWebhookForwardRule        createWebhookForwardRuleUC
	listWebhookForwardRules         listWebhookForwardRulesUC
	deleteWebhookForwardRule        deleteWebhookForwardRuleUC
	listWebhookForwardDeliveries    listWebhookForwardDeliveriesUC
	listProviderWebhooks            listProviderWebhooksUC
	listAllProviderWebhooks         listAllProviderWebhooksUC
	listAvailableWebhookTopics      listAvailableWebhookTopicsUC
	createProviderWebhook           createProviderWebhookUC
	deleteProviderWebhook           deleteProviderWebhookUC
	connectBol                      connectBolUC
	connectWhatsapp                 connectWhatsappUC
	connectStripe                   connectStripeUC
	connectRevolut                  connectRevolutUC
	listPaymentConnections          listPaymentConnectionsUC
	pushProductToBol                pushProductToBolUC
	getBolProductOverride           getBolProductOverrideUC
	upsertBolProductOverride        upsertBolProductOverrideUC
	getBolProductContent            getBolProductContentUC
	upsertBolProductContent         upsertBolProductContentUC
	getBolCommission                getBolCommissionUC
	listBolEconomicOperators        listBolEconomicOperatorsUC
	listWebhookEvents               listWebhookEventsUC
	retryWebhookEventDispatch       retryWebhookEventDispatchUC
	linkExistingWooProduct          linkExistingWooProductUC
	abandonAllWebhookFallbackEvents abandonAllWebhookFallbackEventsUC
	setWhatsAppTestMode             setWhatsAppTestModeUC
	updateWhatsAppCredentials       updateWhatsAppCredentialsUC
	setWhatsAppAIParsing            setWhatsAppAIParsingUC
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
	setSalesChannelPaymentMethod setSalesChannelPaymentMethodUC,
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
	listProviderWebhooks listProviderWebhooksUC,
	listAllProviderWebhooks listAllProviderWebhooksUC,
	listAvailableWebhookTopics listAvailableWebhookTopicsUC,
	createProviderWebhook createProviderWebhookUC,
	deleteProviderWebhook deleteProviderWebhookUC,
	connectBol connectBolUC,
	connectWhatsapp connectWhatsappUC,
	connectStripe connectStripeUC,
	connectRevolut connectRevolutUC,
	listPaymentConnections listPaymentConnectionsUC,
	pushProductToBol pushProductToBolUC,
	getBolProductOverride getBolProductOverrideUC,
	upsertBolProductOverride upsertBolProductOverrideUC,
	getBolProductContent getBolProductContentUC,
	upsertBolProductContent upsertBolProductContentUC,
	getBolCommission getBolCommissionUC,
	listBolEconomicOperators listBolEconomicOperatorsUC,
	listWebhookEvents listWebhookEventsUC,
	retryWebhookEventDispatch retryWebhookEventDispatchUC,
	linkExistingWooProduct linkExistingWooProductUC,
	listProductChannelListings listProductChannelListingsUC,
	abandonAllWebhookFallbackEvents abandonAllWebhookFallbackEventsUC,
	setWhatsAppTestMode setWhatsAppTestModeUC,
	setWhatsAppAIParsing setWhatsAppAIParsingUC,
	updateWhatsAppCredentials updateWhatsAppCredentialsUC,
) *Handler {
	return &Handler{
		listConnections:                 listConnections,
		startWooConnect:                 startWooConnect,
		completeWooConnect:              completeWooConnect,
		manualConnectWoo:                manualConnectWoo,
		disconnect:                      disconnect,
		testConnection:                  testConnection,
		listWooTaxes:                    listWooTaxes,
		createImportJob:                 createImportJob,
		pushProductToWoo:                pushProductToWoo,
		listSyncSchedules:               listSyncSchedules,
		upsertSyncSchedules:             upsertSyncSchedules,
		deleteAllImportedProds:          deleteAllImportedProds,
		updateConnectionSettings:        updateConnectionSettings,
		getProductChannelListing:        getProductChannelListing,
		listSyncLogs:                    listSyncLogs,
		listAllSyncLogs:                 listAllSyncLogs,
		listSyncLogsNeedingAttention:    listSyncLogsNeedingAttention,
		retryFailedSync:                 retryFailedSync,
		dismissFailedSync:               dismissFailedSync,
		resyncProductToChannels:         resyncProductToChannels,
		getAIConnection:                 getAIConnection,
		connectAIProvider:               connectAIProvider,
		disconnectAIConnection:          disconnectAIConnection,
		listSalesChannels:               listSalesChannels,
		createSalesChannel:              createSalesChannel,
		updateSalesChannel:              updateSalesChannel,
		setSalesChannelPaymentMethod:    setSalesChannelPaymentMethod,
		deleteSalesChannel:              deleteSalesChannel,
		listSalesChannelProducts:        listSalesChannelProducts,
		setProductChannelOverride:       setProductChannelOverride,
		fetchWooOrder:                   fetchWooOrder,
		listWooCategories:               listWooCategories,
		listExternalCategoryLinks:       listExternalCategoryLinks,
		upsertExternalCategoryLink:      upsertExternalCategoryLink,
		deleteExternalCategoryLink:      deleteExternalCategoryLink,
		setWebhookSecret:                setWebhookSecret,
		listWebhookFallbackEvents:       listWebhookFallbackEvents,
		retryWebhookFallbackEvent:       retryWebhookFallbackEvent,
		abandonWebhookFallbackEvent:     abandonWebhookFallbackEvent,
		createWebhookForwardRule:        createWebhookForwardRule,
		listWebhookForwardRules:         listWebhookForwardRules,
		deleteWebhookForwardRule:        deleteWebhookForwardRule,
		listWebhookForwardDeliveries:    listWebhookForwardDeliveries,
		listProviderWebhooks:            listProviderWebhooks,
		listAllProviderWebhooks:         listAllProviderWebhooks,
		listAvailableWebhookTopics:      listAvailableWebhookTopics,
		createProviderWebhook:           createProviderWebhook,
		deleteProviderWebhook:           deleteProviderWebhook,
		connectBol:                      connectBol,
		connectWhatsapp:                 connectWhatsapp,
		connectStripe:                   connectStripe,
		connectRevolut:                  connectRevolut,
		listPaymentConnections:          listPaymentConnections,
		pushProductToBol:                pushProductToBol,
		getBolProductOverride:           getBolProductOverride,
		upsertBolProductOverride:        upsertBolProductOverride,
		getBolProductContent:            getBolProductContent,
		upsertBolProductContent:         upsertBolProductContent,
		getBolCommission:                getBolCommission,
		listBolEconomicOperators:        listBolEconomicOperators,
		listWebhookEvents:               listWebhookEvents,
		retryWebhookEventDispatch:       retryWebhookEventDispatch,
		linkExistingWooProduct:          linkExistingWooProduct,
		listProductChannelListings:      listProductChannelListings,
		abandonAllWebhookFallbackEvents: abandonAllWebhookFallbackEvents,
		setWhatsAppTestMode:             setWhatsAppTestMode,
		setWhatsAppAIParsing:            setWhatsAppAIParsing,
		updateWhatsAppCredentials:       updateWhatsAppCredentials,
	}
}

var _ genintegrations.ServerInterface = (*Handler)(nil)
var _ gencallback.ServerInterface = (*Handler)(nil)

// costLedgerRecorderAdapter bridges application/inventory's local
// costLedgerRecorder interface to application/pms's real
// RecordCostLedgerMovementUseCase -- lives in the composition root because
// application/pms already imports application/inventory (see
// application/pms/recalculate_cost_prices.go's doc comment), so the
// reverse import would cycle. Duplicated per handler package, same
// reasoning as infrastructure/handlers/inventory's own copy.
type costLedgerRecorderAdapter struct {
	uc *apppms.RecordCostLedgerMovementUseCase
}

func (a costLedgerRecorderAdapter) Execute(ctx context.Context, in appinventory.CostLedgerMovementInput) error {
	return a.uc.Execute(ctx, apppms.RecordCostLedgerMovementInput{
		SKUID: in.SKUID, QuantityDelta: in.QuantityDelta,
		SourceType:    pmsDomain.SKUCostPriceLedgerSourceType(in.SourceType),
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

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
	webhookEventRepo := repositories.NewWebhookEventRepo(db)
	shipmentRepo := repositories.NewShipmentRepo(db)
	shipmentStatusEventRepo := repositories.NewShipmentStatusEventRepo(db)
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
	exchangeRateRepo := repositories.NewExchangeRateRepo(db)
	channelWarehouseRepo := repositories.NewWarehouseRepo(db)
	salesChannelOverrideRepo := repositories.NewSalesChannelProductOverrideRepo(db)
	mediaRepo := repositories.NewMediaRepo(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	salesOrderLineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	salesOrderLinkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	costLedgerRecorder := costLedgerRecorderAdapter{uc: apppms.NewRecordCostLedgerMovementUseCase(
		db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), appinventory.NewListBatchesUseCase(db, batchRepo, stockMovementRepo, channelWarehouseRepo),
	)}
	stockRecorderUC := appinventory.NewRecordStockMovementUseCase(db, stockMovementRepo, outboxRepo, costLedgerRecorder)
	allocateStockUC := appinventory.NewAllocateSaleStockUseCase(db, stockMovementRepo, batchRepo, outboxRepo)
	updateSalesOrderForWebhooksUC := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, allocateStockUC, stockRecorderUC, outboxRepo)
	retryablePaymentHandlers := map[string]appshipping.RetryableWebhookHandler{
		"stripe":  apppayments.NewWebhookDispatchHandler(db, salesOrderRepo, apppayments.NewStripeEventParser(), updateSalesOrderForWebhooksUC, "stripe"),
		"revolut": apppayments.NewWebhookDispatchHandler(db, salesOrderRepo, apppayments.NewRevolutEventParser(), updateSalesOrderForWebhooksUC, "revolut"),
	}
	fulfilmentOrderRepo := repositories.NewMyParcelFulfilmentOrderRepo(db)
	deleteSalesOrderUC := appoms.NewDeleteSalesOrderUseCase(
		db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, stockRecorderUC, outboxRepo,
	)
	customerRepo := repositories.NewCustomerRepo(db)
	fetchWooOrderUC := appchannels.NewImportWooOrdersUseCase(
		db, connRepo, wooFactory, encryptionSecret,
		salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, customerRepo,
		allocateStockUC, appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, salesOrderLineItemRepo, salesOrderLinkRepo, skuRepo, allocateStockUC, stockRecorderUC, outboxRepo), repositories.NewOutboxRepository(db),
	)

	stockLevelUC := appinventory.NewGetSKUStockLevelUseCase(db, stockMovementRepo, repositories.NewBatchRepo(db), repositories.NewTenantSettingsRepo(db))
	categoryLinkRepo := repositories.NewExternalCategoryLinkRepo(db)
	resolveCategoryLinkUC := appchannels.NewResolveCategoryLinkUseCase(db, categoryLinkRepo, categoryRepo, repositories.NewOutboxRepository(db))
	pushProductToWooUC := appchannels.NewPushProductToWooUseCase(
		db, connRepo, wooFactory, encryptionSecret,
		productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo,
		stockLevelUC,
		mediaRepo,
		resolveCategoryLinkUC, repositories.NewOutboxRepository(db),
	)
	bolOverrideRepo := repositories.NewBolProductOverrideRepo(db)
	bolContentRepo := repositories.NewBolProductContentRepo(db)
	pushProductToBolUC := appchannels.NewPushProductToBolUseCase(
		db, connRepo, bolinfra.NewClientFactory(), encryptionSecret, productRepo, variantRepo, skuRepo, listingRepo, bolOverrideRepo, bolContentRepo, syncLogRepo, stockLevelUC,
		appnotifications.NewDeliverNotificationUseCase(db, notificationRepo, repositories.NewNotificationPreferencesRepo(db), outboxRepo), repositories.NewOutboxRepository(db),
	)
	syncStockToChannelsUC := appchannels.NewSyncStockToChannelsUseCase(
		db, skuRepo, variantRepo, listingRepo, syncLogRepo, connRepo, wooFactory, stockLevelUC, encryptionSecret, pushProductToBolUC, repositories.NewOutboxRepository(db),
	)
	syncProductToChannelsUC := appchannels.NewSyncProductToChannelsUseCase(
		db, productRepo, variantRepo, skuRepo, listingRepo, syncLogRepo, connRepo, wooFactory, encryptionSecret, pushProductToWooUC, mediaRepo, resolveCategoryLinkUC, pushProductToBolUC, pushProductToBolUC, syncStockToChannelsUC, repositories.NewOutboxRepository(db),
	)
	enrollProductInChannelsUC := appchannels.NewEnrollProductInChannelsUseCase(
		db, productRepo, salesChannelRepo, salesChannelOverrideRepo, pushProductToWooUC, pushProductToBolUC,
	)
	resyncProductToChannelsUC := appchannels.NewResyncProductToChannelsUseCase(
		db, variantRepo, skuRepo, syncProductToChannelsUC, syncStockToChannelsUC, enrollProductInChannelsUC,
	)
	listAllProviderWebhooksUC := appintegrations.NewListAllProviderWebhooksUseCase(db, connRepo, wooFactory, myparcel.NewClientFactory(), encryptionSecret)
	notifyShipmentDeliveryUC := appwhatsapp.NewNotifyShipmentDeliveryUseCase(db, salesOrderRepo, customerRepo, connRepo, evolutionapi.NewClientFactory(), encryptionSecret)

	h := NewHandler(
		appintegrations.NewListConnectionsUseCase(db, connRepo),
		appintegrations.NewStartWooConnectUseCase(stateSecret, publicAPIBaseURL, frontendBaseURL),
		appintegrations.NewCompleteWooConnectUseCase(db, connRepo, outboxRepo, stateSecret, encryptionSecret),
		appintegrations.NewManualConnectWooCommerceUseCase(db, connRepo, outboxRepo, encryptionSecret),
		appintegrations.NewDisconnectConnectionUseCase(db, connRepo, outboxRepo),
		appintegrations.NewTestConnectionUseCase(db, connRepo, wooFactory, encryptionSecret, outboxRepo),
		appintegrations.NewListWooTaxesUseCase(db, connRepo, wooFactory, encryptionSecret),
		appchannels.NewCreateImportJobUseCase(
			db, importJobRepo, listingRepo, appnotifications.NewDeliverNotificationUseCase(db, notificationRepo, repositories.NewNotificationPreferencesRepo(db), outboxRepo), repositories.NewOutboxRepository(db),
		),
		pushProductToWooUC,
		appchannels.NewListSyncSchedulesUseCase(db, scheduleRepo),
		appchannels.NewUpsertSyncSchedulesUseCase(db, scheduleRepo, repositories.NewOutboxRepository(db)),
		appchannels.NewDeleteAllImportedProductsUseCase(db, listingRepo, productRepo, categoryRepo, salesOrderRepo, deleteSalesOrderUC, repositories.NewOutboxRepository(db)),
		appintegrations.NewUpdateConnectionSettingsUseCase(db, connRepo, outboxRepo),
		appchannels.NewGetProductChannelListingUseCase(db, listingRepo),
		appchannels.NewListSyncLogsUseCase(db, listingRepo, syncLogRepo),
		appchannels.NewListAllSyncLogsUseCase(db, syncLogRepo, productRepo, connRepo),
		appchannels.NewListSyncLogsNeedingAttentionUseCase(db, syncLogRepo, productRepo, connRepo),
		appchannels.NewRetryFailedSyncUseCase(db, syncLogRepo, pushProductToWooUC),
		appchannels.NewDismissFailedSyncUseCase(db, syncLogRepo, repositories.NewOutboxRepository(db)),
		resyncProductToChannelsUC,
		appintegrations.NewGetAIConnectionUseCase(db, aiConnRepo),
		appintegrations.NewConnectAIProviderUseCase(db, aiConnRepo, encryptionSecret, outboxRepo),
		appintegrations.NewDisconnectAIConnectionUseCase(db, aiConnRepo, outboxRepo),
		appchannels.NewListSalesChannelsUseCase(db, salesChannelRepo),
		appchannels.NewCreateSalesChannelUseCase(db, salesChannelRepo, connRepo, exchangeRateRepo, channelWarehouseRepo, repositories.NewTenantSettingsRepo(db), repositories.NewOutboxRepository(db)),
		appchannels.NewUpdateSalesChannelUseCase(db, salesChannelRepo, connRepo, channelWarehouseRepo, repositories.NewOutboxRepository(db)),
		appchannels.NewSetSalesChannelPaymentMethodUseCase(db, salesChannelRepo, connRepo, repositories.NewOutboxRepository(db)),
		appchannels.NewDeleteSalesChannelUseCase(db, salesChannelRepo, repositories.NewOutboxRepository(db)),
		appchannels.NewListProductsWithChannelEnrollmentUseCase(db, productRepo, salesChannelRepo, salesChannelOverrideRepo),
		appchannels.NewSetProductChannelOverrideUseCase(db, salesChannelOverrideRepo, salesChannelRepo, productRepo, repositories.NewOutboxRepository(db)),
		fetchWooOrderUC,
		appintegrations.NewListWooCategoriesUseCase(db, connRepo, wooFactory, encryptionSecret),
		appchannels.NewListExternalCategoryLinksUseCase(db, categoryLinkRepo),
		appchannels.NewUpsertExternalCategoryLinkUseCase(db, categoryLinkRepo, categoryRepo, repositories.NewOutboxRepository(db)),
		appchannels.NewDeleteExternalCategoryLinkUseCase(db, categoryLinkRepo, repositories.NewOutboxRepository(db)),
		appintegrations.NewSetWebhookSecretUseCase(db, connRepo, encryptionSecret, outboxRepo),
		appwebhooks.NewListWebhookFallbackEventsUseCase(db, webhookFallbackRepo),
		appwebhooks.NewRetryWebhookFallbackEventUseCase(db, webhookFallbackRepo, webhookPublisher, outboxRepo),
		appwebhooks.NewAbandonWebhookFallbackEventUseCase(db, webhookFallbackRepo, outboxRepo),
		appwebhooks.NewCreateForwardRuleUseCase(db, webhookForwardRuleRepo, outboxRepo),
		appwebhooks.NewListForwardRulesUseCase(db, webhookForwardRuleRepo),
		appwebhooks.NewDeleteForwardRuleUseCase(db, webhookForwardRuleRepo, outboxRepo),
		appwebhooks.NewListForwardDeliveriesUseCase(db, webhookForwardDeliveryRepo),
		appintegrations.NewListProviderWebhooksUseCase(db, connRepo, wooFactory, myparcel.NewClientFactory(), encryptionSecret),
		listAllProviderWebhooksUC,
		appwebhooks.NewListAvailableWebhookTopicsUseCase(db, webhookEventRepo, listAllProviderWebhooksUC),
		appintegrations.NewCreateProviderWebhookUseCase(db, connRepo, wooFactory, myparcel.NewClientFactory(), encryptionSecret, publicAPIBaseURL, outboxRepo),
		appintegrations.NewDeleteProviderWebhookUseCase(db, connRepo, wooFactory, myparcel.NewClientFactory(), encryptionSecret, outboxRepo),
		appintegrations.NewConnectBolUseCase(db, connRepo, outboxRepo, encryptionSecret),
		appintegrations.NewConnectWhatsAppUseCase(db, connRepo, outboxRepo, encryptionSecret, publicAPIBaseURL),
		appintegrations.NewConnectStripeUseCase(db, connRepo, outboxRepo, encryptionSecret, publicAPIBaseURL),
		appintegrations.NewConnectRevolutUseCase(db, connRepo, outboxRepo, encryptionSecret, publicAPIBaseURL),
		appintegrations.NewListPaymentConnectionsUseCase(db, connRepo),
		pushProductToBolUC,
		appchannels.NewGetBolProductOverrideUseCase(db, bolOverrideRepo),
		appchannels.NewUpsertBolProductOverrideUseCase(db, bolOverrideRepo, outboxRepo),
		appchannels.NewGetBolProductContentUseCase(db, bolContentRepo),
		appchannels.NewUpsertBolProductContentUseCase(db, bolContentRepo, outboxRepo),
		appchannels.NewGetBolCommissionUseCase(db, connRepo, bolinfra.NewClientFactory(), encryptionSecret, skuRepo, bolOverrideRepo),
		appintegrations.NewListBolEconomicOperatorsUseCase(db, connRepo, bolinfra.NewClientFactory(), encryptionSecret),
		appwebhooks.NewListWebhookEventsUseCase(db, webhookEventRepo),
		appshipping.NewRetryWebhookEventDispatchUseCase(db, webhookEventRepo, shipmentRepo, shipmentStatusEventRepo, connRepo, salesOrderRepo, myparcel.NewCarrierClientFactory(), encryptionSecret, fulfilmentOrderRepo, updateSalesOrderForWebhooksUC, notifyShipmentDeliveryUC, outboxRepo, retryablePaymentHandlers),
		appchannels.NewLinkExistingWooProductUseCase(db, productRepo, connRepo, listingRepo, wooFactory, encryptionSecret, repositories.NewOutboxRepository(db)),
		appchannels.NewListProductChannelListingsUseCase(db, listingRepo),
		appwebhooks.NewAbandonAllWebhookFallbackEventsUseCase(db, webhookFallbackRepo, outboxRepo),
		appintegrations.NewSetWhatsAppTestModeUseCase(db, connRepo, outboxRepo),
		appintegrations.NewSetWhatsAppAIParsingUseCase(db, connRepo, aiConnRepo, outboxRepo),
		appintegrations.NewUpdateWhatsAppCredentialsUseCase(db, connRepo, encryptionSecret, outboxRepo),
	)
	RegisterRoutes(rg, h)
	RegisterPublicRoutes(root, h)
}
