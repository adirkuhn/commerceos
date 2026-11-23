package integrations

import (
	"context"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
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
	Execute(ctx context.Context, connectionID string, providedSecret string) (*appintegrations.SetWebhookSecretResult, error)
}

type listWebhookFallbackEventsUC interface {
	Execute(ctx context.Context, status webhooksDomain.FallbackStatus, page, perPage int) (*appwebhooks.ListWebhookFallbackEventsResult, error)
}

type retryWebhookFallbackEventUC interface {
	Execute(ctx context.Context, id string) error
}

type abandonWebhookFallbackEventUC interface {
	Execute(ctx context.Context, id string) error
}

type setWhatsAppAIParsingUC interface {
	Execute(ctx context.Context, connectionID string, enabled bool) (*integrationsDomain.Connection, error)
}

type setWhatsAppTestModeUC interface {
	Execute(ctx context.Context, connectionID string, enabled bool, testPhoneE164 string) (*integrationsDomain.Connection, error)
}

type updateWhatsAppCredentialsUC interface {
	Execute(ctx context.Context, connectionID string, in appintegrations.UpdateWhatsAppCredentialsInput) (*integrationsDomain.Connection, error)
}

type abandonAllWebhookFallbackEventsUC interface {
	Execute(ctx context.Context, status webhooksDomain.FallbackStatus) (int64, error)
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

type listWebhookEventsUC interface {
	Execute(ctx context.Context, provider string, limit int) ([]webhooksDomain.WebhookEvent, error)
}

type retryWebhookEventDispatchUC interface {
	Execute(ctx context.Context, eventID string) (*webhooksDomain.WebhookEvent, error)
}

type listProviderWebhooksUC interface {
	Execute(ctx context.Context, connectionID string) ([]integrationsDomain.ProviderWebhook, error)
}

type listAllProviderWebhooksUC interface {
	Execute(ctx context.Context) (*appintegrations.ListAllProviderWebhooksResult, error)
}

type listAvailableWebhookTopicsUC interface {
	Execute(ctx context.Context) (*appwebhooks.AvailableWebhookTopics, error)
}

type createProviderWebhookUC interface {
	Execute(ctx context.Context, connectionID, topic string) (*integrationsDomain.ProviderWebhook, error)
}

type deleteProviderWebhookUC interface {
	Execute(ctx context.Context, connectionID, webhookID string) error
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

type listProductChannelListingsUC interface {
	Execute(ctx context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error)
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

type linkExistingWooProductUC interface {
	Execute(ctx context.Context, in appchannels.LinkExistingWooProductInput) (*channelsDomain.ProductChannelListing, error)
}

type dismissFailedSyncUC interface {
	Execute(ctx context.Context, syncLogEntryID string) error
}

type listAIConnectionsUC interface {
	Execute(ctx context.Context) ([]integrationsDomain.AIConnection, error)
}

type createAIConnectionUC interface {
	Execute(ctx context.Context, in appintegrations.AIConnectionInput) (*integrationsDomain.AIConnection, error)
}

type updateAIConnectionUC interface {
	Execute(ctx context.Context, id string, p appintegrations.AIConnectionPatch) (*integrationsDomain.AIConnection, error)
}

type deleteAIConnectionUC interface {
	Execute(ctx context.Context, id string) error
}

type reorderAIConnectionsUC interface {
	Execute(ctx context.Context, ids []string) ([]integrationsDomain.AIConnection, error)
}

// AIConnectionUseCases are the ordered AI connection list's operations.
type AIConnectionUseCases struct {
	List    listAIConnectionsUC
	Create  createAIConnectionUC
	Update  updateAIConnectionUC
	Delete  deleteAIConnectionUC
	Reorder reorderAIConnectionsUC
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

type setSalesChannelPaymentMethodUC interface {
	Execute(ctx context.Context, channelID string, paymentConnectionID *string) (*channelsDomain.SalesChannel, error)
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

type connectWhatsappUC interface {
	Execute(ctx context.Context, in appintegrations.ConnectWhatsAppInput) (*appintegrations.ConnectWhatsAppResult, error)
}

type connectStripeUC interface {
	Execute(ctx context.Context, in appintegrations.ConnectStripeInput) (*appintegrations.ConnectStripeResult, error)
}

type connectRevolutUC interface {
	Execute(ctx context.Context, in appintegrations.ConnectRevolutInput) (*appintegrations.ConnectRevolutResult, error)
}

type listPaymentConnectionsUC interface {
	Execute(ctx context.Context) ([]integrationsDomain.Connection, error)
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

type previewBolPricingUC interface {
	Execute(ctx context.Context, in appchannels.PreviewBolPricingInput) (*appchannels.BolPricingPreview, error)
}

type listBolEconomicOperatorsUC interface {
	Execute(ctx context.Context, connectionID string) ([]integrationsDomain.BolEconomicOperator, error)
}
