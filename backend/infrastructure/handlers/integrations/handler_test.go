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
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
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
	out      []channelsDomain.SyncSchedule
	err      error
	gotInput appchannels.UpsertSyncSchedulesInput
}

func (s *stubUpsertSyncSchedule) Execute(_ context.Context, in appchannels.UpsertSyncSchedulesInput) ([]channelsDomain.SyncSchedule, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	out := make([]channelsDomain.SyncSchedule, len(in.Schedules))
	for i, sc := range in.Schedules {
		out[i] = channelsDomain.SyncSchedule{
			EntityType: sc.EntityType, Direction: sc.Direction, Enabled: sc.Enabled, IntervalMinutes: sc.IntervalMinutes,
		}
	}
	return out, nil
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

type stubListSyncLogsNeedingAttention struct {
	out   []appchannels.SyncLogEntryWithNames
	total int64
	err   error
}

func (s *stubListSyncLogsNeedingAttention) Execute(_ context.Context, _, _ int) ([]appchannels.SyncLogEntryWithNames, int64, error) {
	return s.out, s.total, s.err
}

type stubRetryFailedSync struct {
	err   error
	gotID string
}

func (s *stubRetryFailedSync) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubConnectBol struct {
	out *integrationsDomain.Connection
	err error
}

func (s *stubConnectBol) Execute(_ context.Context, _ appintegrations.ConnectBolInput) (*integrationsDomain.Connection, error) {
	return s.out, s.err
}

type stubPushProductToBol struct {
	out []channelsDomain.ProductChannelListing
	err error
}

func (s *stubPushProductToBol) Execute(_ context.Context, _ appchannels.PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error) {
	return s.out, s.err
}

type stubGetBolProductOverride struct {
	out *channelsDomain.BolProductOverride
	err error
}

func (s *stubGetBolProductOverride) Execute(_ context.Context, _, _ string) (*channelsDomain.BolProductOverride, error) {
	return s.out, s.err
}

type stubUpsertBolProductOverride struct {
	out      *channelsDomain.BolProductOverride
	err      error
	gotInput appchannels.UpsertBolProductOverrideInput
}

func (s *stubUpsertBolProductOverride) Execute(_ context.Context, in appchannels.UpsertBolProductOverrideInput) (*channelsDomain.BolProductOverride, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubGetBolProductContent struct {
	out *channelsDomain.BolProductContent
	err error
}

func (s *stubGetBolProductContent) Execute(_ context.Context, _, _ string) (*channelsDomain.BolProductContent, error) {
	return s.out, s.err
}

type stubUpsertBolProductContent struct {
	out      *channelsDomain.BolProductContent
	err      error
	gotInput appchannels.UpsertBolProductContentInput
}

func (s *stubUpsertBolProductContent) Execute(_ context.Context, in appchannels.UpsertBolProductContentInput) (*channelsDomain.BolProductContent, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubGetBolCommission struct {
	out *appchannels.BolCommissionResult
	err error
}

func (s *stubGetBolCommission) Execute(_ context.Context, _ appchannels.GetBolCommissionInput) (*appchannels.BolCommissionResult, error) {
	return s.out, s.err
}

type stubListBolEconomicOperators struct {
	out []integrationsDomain.BolEconomicOperator
	err error
}

func (s *stubListBolEconomicOperators) Execute(_ context.Context, _ string) ([]integrationsDomain.BolEconomicOperator, error) {
	return s.out, s.err
}

type stubDismissFailedSync struct {
	err   error
	gotID string
}

func (s *stubDismissFailedSync) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubResyncProductToChannels struct {
	err          error
	gotProductID string
}

func (s *stubResyncProductToChannels) Execute(_ context.Context, productID string) error {
	s.gotProductID = productID
	return s.err
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

type stubFetchWooOrder struct {
	out *appchannels.FetchWooOrderResult
	err error
	in  appchannels.FetchWooOrderInput
}

func (s *stubFetchWooOrder) ExecuteOne(_ context.Context, in appchannels.FetchWooOrderInput) (*appchannels.FetchWooOrderResult, error) {
	s.in = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appchannels.FetchWooOrderResult{ExternalOrderID: in.ExternalOrderID}, nil
}

type stubListWooCategories struct {
	out []integrationsDomain.WooCategory
	err error
}

func (s *stubListWooCategories) Execute(_ context.Context, _ string) ([]integrationsDomain.WooCategory, error) {
	return s.out, s.err
}

type stubListExternalCategoryLinks struct {
	out []channelsDomain.ExternalCategoryLink
	err error
}

func (s *stubListExternalCategoryLinks) Execute(_ context.Context, _ string) ([]channelsDomain.ExternalCategoryLink, error) {
	return s.out, s.err
}

type stubUpsertExternalCategoryLink struct {
	out *channelsDomain.ExternalCategoryLink
	err error
	in  appchannels.UpsertExternalCategoryLinkInput
}

func (s *stubUpsertExternalCategoryLink) Execute(_ context.Context, in appchannels.UpsertExternalCategoryLinkInput) (*channelsDomain.ExternalCategoryLink, error) {
	s.in = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &channelsDomain.ExternalCategoryLink{ID: "link-1", ConnectionID: "", ExternalID: in.ExternalID, CategoryID: in.CategoryID}, nil
}

type stubDeleteExternalCategoryLink struct {
	err error
	in  string
}

func (s *stubDeleteExternalCategoryLink) Execute(_ context.Context, id string) error {
	s.in = id
	return s.err
}

type stubSetWebhookSecret struct {
	out *appintegrations.SetWebhookSecretResult
	err error
}

func (s *stubSetWebhookSecret) Execute(_ context.Context, _ string) (*appintegrations.SetWebhookSecretResult, error) {
	return s.out, s.err
}

type stubListWebhookFallbackEvents struct {
	out []webhooksDomain.FallbackEvent
	err error
}

func (s *stubListWebhookFallbackEvents) Execute(_ context.Context, _ webhooksDomain.FallbackStatus) ([]webhooksDomain.FallbackEvent, error) {
	return s.out, s.err
}

type stubRetryWebhookFallbackEvent struct{ err error }

func (s *stubRetryWebhookFallbackEvent) Execute(_ context.Context, _ string) error { return s.err }

type stubAbandonWebhookFallbackEvent struct{ err error }

func (s *stubAbandonWebhookFallbackEvent) Execute(_ context.Context, _ string) error { return s.err }

type stubCreateWebhookForwardRule struct {
	out *webhooksDomain.ForwardRule
	err error
}

func (s *stubCreateWebhookForwardRule) Execute(_ context.Context, provider, topic, targetURL string) (*webhooksDomain.ForwardRule, error) {
	if s.out != nil {
		return s.out, s.err
	}
	return &webhooksDomain.ForwardRule{ID: "rule-1", Provider: provider, Topic: topic, TargetURL: targetURL, Active: true}, s.err
}

type stubListWebhookForwardRules struct {
	out []webhooksDomain.ForwardRule
	err error
}

func (s *stubListWebhookForwardRules) Execute(_ context.Context) ([]webhooksDomain.ForwardRule, error) {
	return s.out, s.err
}

type stubDeleteWebhookForwardRule struct{ err error }

func (s *stubDeleteWebhookForwardRule) Execute(_ context.Context, _ string) error { return s.err }

type stubListWebhookForwardDeliveries struct {
	out []webhooksDomain.ForwardDelivery
	err error
}

func (s *stubListWebhookForwardDeliveries) Execute(_ context.Context, _ int) ([]webhooksDomain.ForwardDelivery, error) {
	return s.out, s.err
}

type stubListProviderWebhooks struct {
	out []integrationsDomain.ProviderWebhook
	err error
}

func (s *stubListProviderWebhooks) Execute(_ context.Context, _ string) ([]integrationsDomain.ProviderWebhook, error) {
	return s.out, s.err
}

type stubListAllProviderWebhooks struct {
	out *appintegrations.ListAllProviderWebhooksResult
	err error
}

func (s *stubListAllProviderWebhooks) Execute(_ context.Context) (*appintegrations.ListAllProviderWebhooksResult, error) {
	if s.out != nil {
		return s.out, s.err
	}
	return &appintegrations.ListAllProviderWebhooksResult{}, s.err
}

type stubCreateProviderWebhook struct {
	out *integrationsDomain.ProviderWebhook
	err error
}

func (s *stubCreateProviderWebhook) Execute(_ context.Context, connectionID, topic string) (*integrationsDomain.ProviderWebhook, error) {
	if s.out != nil {
		return s.out, s.err
	}
	return &integrationsDomain.ProviderWebhook{ID: "hook-1", Provider: integrationsDomain.ProviderWooCommerce, Topic: topic}, s.err
}

type stubDeleteProviderWebhook struct{ err error }

func (s *stubDeleteProviderWebhook) Execute(_ context.Context, _, _ string) error { return s.err }

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
		&stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
}

// newSyncScheduleTestHandler builds a Handler for the sync-schedule routes
// specifically — every other use case is a harmless zero-value stub, same
// convention as newTestHandler's own hardcoded importWooProducts/pushProductToWoo.
func newSyncScheduleTestHandler(list *stubListSyncSchedules, upsert *stubUpsertSyncSchedule) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		list, upsert, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newListWooTaxesTestHandler builds a Handler for the woo-taxes route
// specifically — every other use case is a harmless zero-value stub.
func newListWooTaxesTestHandler(listWooTaxes *stubListWooTaxes) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, listWooTaxes, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newCategoryLinksTestHandler builds a Handler for the woo-categories/
// category-links routes specifically — every other use case is a harmless
// zero-value stub.
func newCategoryLinksTestHandler(listWooCategories *stubListWooCategories, listLinks *stubListExternalCategoryLinks, upsert *stubUpsertExternalCategoryLink, del *stubDeleteExternalCategoryLink) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{},
		listWooCategories, listLinks, upsert, del, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newDeleteImportedTestHandler builds a Handler for the delete-all-imported
// route specifically — every other use case is a harmless zero-value stub.
func newDeleteImportedTestHandler(deleteAll *stubDeleteAllImportedProducts) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, deleteAll, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newUpdateConnectionSettingsTestHandler builds a Handler for the
// connection-settings route specifically.
func newUpdateConnectionSettingsTestHandler(settings *stubUpdateConnectionSettings) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, settings, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newGetListingTestHandler builds a Handler for the get-product-channel-listing
// route specifically — every other use case is a harmless zero-value stub.
func newGetListingTestHandler(getListing *stubGetProductChannelListing) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		getListing, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newListSyncLogsTestHandler builds a Handler for the list-sync-logs route
// specifically — every other use case is a harmless zero-value stub.
func newListSyncLogsTestHandler(listLogs *stubListSyncLogs) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, listLogs, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
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
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, listAll, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newSyncLogAttentionTestHandler builds a Handler for the "needs attention"/
// retry/dismiss sync-log routes specifically — every other use case is a
// harmless zero-value stub.
func newSyncLogAttentionTestHandler(needingAttn *stubListSyncLogsNeedingAttention, retry *stubRetryFailedSync, dismiss *stubDismissFailedSync) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{},
		needingAttn, retry, dismiss, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newResyncProductTestHandler builds a Handler for the resync-to-channels
// route specifically — every other use case is a harmless zero-value stub.
func newResyncProductTestHandler(resync *stubResyncProductToChannels) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{},
		resync,
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

func TestResyncProductToChannelsHandler_success(t *testing.T) {
	resync := &stubResyncProductToChannels{}
	h := newResyncProductTestHandler(resync)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/products/product-1/resync", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "product-1", resync.gotProductID)
}

func TestResyncProductToChannelsHandler_notFound(t *testing.T) {
	resync := &stubResyncProductToChannels{err: pmsDomain.ErrProductNotFound}
	h := newResyncProductTestHandler(resync)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/products/missing/resync", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// newFetchWooOrderTestHandler builds a Handler for the fetch-order route
// specifically — every other use case is a harmless zero-value stub.
func newFetchWooOrderTestHandler(fetch *stubFetchWooOrder) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
		fetch, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newBolTestHandler builds a Handler for the Bol routes specifically —
// every other use case is a harmless zero-value stub.
func newBolTestHandler(connectBol *stubConnectBol, pushBol *stubPushProductToBol, getOverride *stubGetBolProductOverride, upsertOverride *stubUpsertBolProductOverride, getContent *stubGetBolProductContent, upsertContent *stubUpsertBolProductContent, getCommission *stubGetBolCommission, listEconomicOperators *stubListBolEconomicOperators) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{},
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{},
		&stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{},
		connectBol, pushBol, getOverride, upsertOverride, getContent, upsertContent, getCommission, listEconomicOperators,
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

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", map[string]any{
		"key_id":          1,
		"user_id":         "some-state-token",
		"consumer_key":    "ck_abc",
		"consumer_secret": "cs_abc",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWooCallbackHandler_acceptsNumericKeyID is the exact production bug:
// WooCommerce sends key_id as a JSON number (the API key row's numeric DB
// id) -- binding it as *string rejected every real callback with a 422
// before key_id was widened to an open (typeless) schema. The field is
// unused past binding either way, so the fix accepts any shape rather
// than pinning to one JSON type.
func TestWooCallbackHandler_acceptsNumericKeyID(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Status: integrationsDomain.StatusConnected}
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{out: conn}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterPublicRoutes(r, h)

	body := []byte(`{"key_id":1,"user_id":"some-state-token","consumer_key":"ck_abc","consumer_secret":"cs_abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestWooCallbackHandler_acceptsStringKeyID proves the open schema didn't
// just flip which single type is accepted -- a string key_id (whatever
// shape sent it that way) must keep working too.
func TestWooCallbackHandler_acceptsStringKeyID(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", Status: integrationsDomain.StatusConnected}
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{out: conn}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterPublicRoutes(r, h)

	body := []byte(`{"key_id":"1","user_id":"some-state-token","consumer_key":"ck_abc","consumer_secret":"cs_abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWooCallbackHandler_invalidState_returns401(t *testing.T) {
	h := newTestHandler(&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{err: integrationsDomain.ErrStateInvalid}, &stubManualConnectWooCommerce{}, &stubDisconnectConnection{}, &stubTestConnection{})
	r := gin.New()
	integrationshandler.RegisterPublicRoutes(r, h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", map[string]any{
		"key_id":          1,
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

	w := doRequest(engine, http.MethodPost, "/api/v1/integrations/woocommerce/connect/callback", map[string]any{
		"key_id": 1, "user_id": "s", "consumer_key": "k", "consumer_secret": "v",
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

func TestListWooCategoriesHandler_returnsCategories(t *testing.T) {
	h := newCategoryLinksTestHandler(&stubListWooCategories{out: []integrationsDomain.WooCategory{
		{ExternalID: "20", Name: "Beverages", Slug: "beverages"},
	}}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/woo-categories", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Beverages")
}

func TestListWooCategoriesHandler_notFound(t *testing.T) {
	h := newCategoryLinksTestHandler(&stubListWooCategories{err: integrationsDomain.ErrConnectionNotFound}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/missing/woo-categories", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListExternalCategoryLinksHandler_returnsLinks(t *testing.T) {
	h := newCategoryLinksTestHandler(&stubListWooCategories{}, &stubListExternalCategoryLinks{out: []channelsDomain.ExternalCategoryLink{
		{ID: "link-1", ConnectionID: "conn-1", ExternalID: "20", CategoryID: "cat-1"},
	}}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/category-links", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "link-1")
}

func TestUpsertExternalCategoryLinkHandler_savesLink(t *testing.T) {
	upsert := &stubUpsertExternalCategoryLink{}
	h := newCategoryLinksTestHandler(&stubListWooCategories{}, &stubListExternalCategoryLinks{}, upsert, &stubDeleteExternalCategoryLink{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/category-links",
		genintegrations.UpsertExternalCategoryLinkRequest{ExternalId: "20", CategoryId: "cat-1"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "conn-1", upsert.in.ConnectionID)
	assert.Equal(t, "20", upsert.in.ExternalID)
	assert.Equal(t, "cat-1", upsert.in.CategoryID)
}

func TestUpsertExternalCategoryLinkHandler_notFound(t *testing.T) {
	h := newCategoryLinksTestHandler(&stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{err: pmsDomain.ErrCategoryNotFound}, &stubDeleteExternalCategoryLink{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/category-links",
		genintegrations.UpsertExternalCategoryLinkRequest{ExternalId: "20", CategoryId: "missing"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteExternalCategoryLinkHandler_deletesLink(t *testing.T) {
	del := &stubDeleteExternalCategoryLink{}
	h := newCategoryLinksTestHandler(&stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, del)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/category-links/link-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "link-1", del.in)
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

func TestUpsertSyncSchedulesHandler_sendsTheWholeBatchInOneCall(t *testing.T) {
	upsert := &stubUpsertSyncSchedule{}
	h := newSyncScheduleTestHandler(&stubListSyncSchedules{}, upsert)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/connections/conn-1/sync-schedules", map[string]any{
		"schedules": []map[string]any{
			{"entity_type": "products", "direction": "import", "enabled": false, "interval_minutes": 60, "last_run_status": "", "last_run_error": ""},
			{"entity_type": "stock", "direction": "export", "enabled": true, "interval_minutes": 30, "last_run_status": "", "last_run_error": ""},
			{"entity_type": "orders", "direction": "import", "enabled": false, "interval_minutes": 60, "last_run_status": "", "last_run_error": ""},
		},
	})

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, upsert.gotInput.Schedules, 3, "the whole grid is submitted as one batch, not looped one row at a time")
	assert.Equal(t, channelsDomain.SyncEntityOrders, upsert.gotInput.Schedules[2].EntityType)
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
		DeletedSalesOrders: 3, DeletedProducts: 805, DeletedCategories: 12,
	}})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/imported-products", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"deleted_sales_orders":3`)
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
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
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
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		get, connect, disconnect,
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
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

func TestFetchWooOrderHandler_success(t *testing.T) {
	fetch := &stubFetchWooOrder{out: &appchannels.FetchWooOrderResult{
		ExternalOrderID: "501", SalesOrderID: "sales-order-1", Created: true,
	}}
	h := newFetchWooOrderTestHandler(fetch)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/fetch-order", map[string]any{
		"external_order_id": "501",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sales_order_id":"sales-order-1"`)
	assert.Contains(t, w.Body.String(), `"created":true`)
	assert.Equal(t, "conn-1", fetch.in.ConnectionID)
	assert.Equal(t, "501", fetch.in.ExternalOrderID)
}

func TestFetchWooOrderHandler_rejectsAMissingExternalOrderID(t *testing.T) {
	h := newFetchWooOrderTestHandler(&stubFetchWooOrder{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/fetch-order", map[string]any{})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFetchWooOrderHandler_propagatesAWooRequestFailure(t *testing.T) {
	h := newFetchWooOrderTestHandler(&stubFetchWooOrder{err: integrationsDomain.ErrWooRequestFailed})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/fetch-order", map[string]any{
		"external_order_id": "999999",
	})

	assert.Equal(t, http.StatusBadGateway, w.Code)
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
	listingID := "listing-1"
	h := newListSyncLogsTestHandler(&stubListSyncLogs{out: []channelsDomain.SyncLogEntry{
		{
			ID: "log-1", ListingID: &listingID, ProductID: "prod-1", ConnectionID: "conn-1",
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
		&stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		list, create, update, del, listProducts, setOverride, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{}, &stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{}, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
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

func TestListSyncLogsNeedingAttentionHandler_returnsFlaggedListings(t *testing.T) {
	needingAttn := &stubListSyncLogsNeedingAttention{out: []appchannels.SyncLogEntryWithNames{
		{SyncLogEntry: channelsDomain.SyncLogEntry{ID: "log-1", ProductID: "prod-1", Status: channelsDomain.SyncLogStatusError}, ProductName: "Widget A"},
	}, total: 1}
	h := newSyncLogAttentionTestHandler(needingAttn, &stubRetryFailedSync{}, &stubDismissFailedSync{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/sync-logs/needing-attention", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Widget A")
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestRetryFailedSyncHandler_success(t *testing.T) {
	retry := &stubRetryFailedSync{}
	h := newSyncLogAttentionTestHandler(&stubListSyncLogsNeedingAttention{}, retry, &stubDismissFailedSync{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sync-logs/log-1/retry", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "log-1", retry.gotID)
}

func TestRetryFailedSyncHandler_notFound(t *testing.T) {
	retry := &stubRetryFailedSync{err: channelsDomain.ErrSyncLogNotFound}
	h := newSyncLogAttentionTestHandler(&stubListSyncLogsNeedingAttention{}, retry, &stubDismissFailedSync{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sync-logs/missing/retry", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRetryFailedSyncHandler_rejectsAnEntryWithNoListing(t *testing.T) {
	retry := &stubRetryFailedSync{err: channelsDomain.ErrSyncLogHasNoListing}
	h := newSyncLogAttentionTestHandler(&stubListSyncLogsNeedingAttention{}, retry, &stubDismissFailedSync{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sync-logs/log-1/retry", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDismissFailedSyncHandler_success(t *testing.T) {
	dismiss := &stubDismissFailedSync{}
	h := newSyncLogAttentionTestHandler(&stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, dismiss)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sync-logs/log-1/dismiss", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "log-1", dismiss.gotID)
}

func TestDismissFailedSyncHandler_notFound(t *testing.T) {
	dismiss := &stubDismissFailedSync{err: channelsDomain.ErrSyncLogNotFound}
	h := newSyncLogAttentionTestHandler(&stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, dismiss)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/sync-logs/missing/dismiss", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// newWebhookAdminTestHandler builds a Handler for the webhook-secret/
// fallback-event admin routes specifically — every other use case is a
// harmless zero-value stub.
func newWebhookAdminTestHandler(setSecret *stubSetWebhookSecret, list *stubListWebhookFallbackEvents, retry *stubRetryWebhookFallbackEvent, abandon *stubAbandonWebhookFallbackEvent) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{},
		setSecret, list, retry, abandon, &stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{}, &stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{}, &stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

// newWebhookForwardTestHandler wires only the four webhook fan-out use
// cases specifically -- every other use case is a harmless zero-value stub.
func newWebhookForwardTestHandler(create *stubCreateWebhookForwardRule, list *stubListWebhookForwardRules, del *stubDeleteWebhookForwardRule, listDeliveries *stubListWebhookForwardDeliveries) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{},
		&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{},
		create, list, del, listDeliveries,
		&stubListProviderWebhooks{}, &stubListAllProviderWebhooks{}, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{},
		&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

func TestCreateWebhookForwardRuleHandler_success(t *testing.T) {
	h := newWebhookForwardTestHandler(&stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/webhook-forward-rules", map[string]any{
		"provider": "woocommerce", "topic": "order.updated", "target_url": "https://example.com/hook",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "rule-1")
	assert.Contains(t, w.Body.String(), "https://example.com/hook")
}

func TestCreateWebhookForwardRuleHandler_invalidTargetURLReturns422(t *testing.T) {
	stub := &stubCreateWebhookForwardRule{err: webhooksDomain.ErrInvalidTargetURL}
	h := newWebhookForwardTestHandler(stub, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/webhook-forward-rules", map[string]any{
		"provider": "woocommerce", "target_url": "not-a-url",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListWebhookForwardRulesHandler_returnsList(t *testing.T) {
	stub := &stubListWebhookForwardRules{out: []webhooksDomain.ForwardRule{{ID: "rule-1", Provider: "woocommerce", TargetURL: "https://example.com/hook", Active: true}}}
	h := newWebhookForwardTestHandler(&stubCreateWebhookForwardRule{}, stub, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/webhook-forward-rules", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "rule-1")
}

func TestDeleteWebhookForwardRuleHandler_success(t *testing.T) {
	stub := &stubDeleteWebhookForwardRule{}
	h := newWebhookForwardTestHandler(&stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, stub, &stubListWebhookForwardDeliveries{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/webhook-forward-rules/rule-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestListWebhookForwardDeliveriesHandler_returnsList(t *testing.T) {
	stub := &stubListWebhookForwardDeliveries{out: []webhooksDomain.ForwardDelivery{{ID: "d-1", Status: webhooksDomain.ForwardDeliveryStatusDelivered}}}
	h := newWebhookForwardTestHandler(&stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/webhook-forward-deliveries", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "d-1")
}

// newProviderWebhookTestHandler wires only the three provider-webhooks use
// cases specifically -- every other use case is a harmless zero-value stub.
func newProviderWebhookTestHandler(list *stubListProviderWebhooks, create *stubCreateProviderWebhook, del *stubDeleteProviderWebhook) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{},
		&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{},
		&stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{},
		list, &stubListAllProviderWebhooks{}, create, del,
		&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

func newListAllProviderWebhooksTestHandler(stub *stubListAllProviderWebhooks) *integrationshandler.Handler {
	return integrationshandler.NewHandler(
		&stubListConnections{}, &stubStartWooConnect{}, &stubCompleteWooConnect{}, &stubManualConnectWooCommerce{},
		&stubDisconnectConnection{}, &stubTestConnection{}, &stubListWooTaxes{}, &stubCreateImportJob{}, &stubPushProductToWoo{},
		&stubListSyncSchedules{}, &stubUpsertSyncSchedule{}, &stubDeleteAllImportedProducts{}, &stubUpdateConnectionSettings{}, &stubGetProductChannelListing{}, &stubListSyncLogs{}, &stubListAllSyncLogs{}, &stubListSyncLogsNeedingAttention{}, &stubRetryFailedSync{}, &stubDismissFailedSync{}, &stubResyncProductToChannels{},
		&stubGetAIConnection{}, &stubConnectAIProvider{}, &stubDisconnectAIConnection{},
		&stubListSalesChannels{}, &stubCreateSalesChannel{}, &stubUpdateSalesChannel{}, &stubDeleteSalesChannel{}, &stubListSalesChannelProducts{}, &stubSetProductChannelOverride{}, &stubFetchWooOrder{}, &stubListWooCategories{}, &stubListExternalCategoryLinks{}, &stubUpsertExternalCategoryLink{}, &stubDeleteExternalCategoryLink{},
		&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{},
		&stubCreateWebhookForwardRule{}, &stubListWebhookForwardRules{}, &stubDeleteWebhookForwardRule{}, &stubListWebhookForwardDeliveries{},
		&stubListProviderWebhooks{}, stub, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{},
		&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{},
	)
}

func TestListAllProviderWebhooksHandler_returnsList(t *testing.T) {
	stub := &stubListAllProviderWebhooks{out: &appintegrations.ListAllProviderWebhooksResult{
		Webhooks: []integrationsDomain.ProviderWebhook{
			{ID: "1", Provider: integrationsDomain.ProviderWooCommerce, Topic: "order.updated", DeliveryURL: "https://a.example.com", Status: "active", ConnectionID: "conn-1", StoreLabel: "https://mystore.com"},
		},
	}}
	h := newListAllProviderWebhooksTestHandler(stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/provider-webhooks", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "order.updated")
	assert.Contains(t, w.Body.String(), "mystore.com")
}

func TestListProviderWebhooksHandler_returnsList(t *testing.T) {
	stub := &stubListProviderWebhooks{out: []integrationsDomain.ProviderWebhook{
		{ID: "1", Provider: integrationsDomain.ProviderWooCommerce, Topic: "order.updated", DeliveryURL: "https://a.example.com", Status: "active"},
	}}
	h := newProviderWebhookTestHandler(stub, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/provider-webhooks", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "order.updated")
}

func TestListProviderWebhooksHandler_unsupportedProvider_returns422(t *testing.T) {
	stub := &stubListProviderWebhooks{err: integrationsDomain.ErrProviderNotSupported}
	h := newProviderWebhookTestHandler(stub, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/provider-webhooks", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListProviderWebhooksHandler_liveCallFailure_returns502(t *testing.T) {
	stub := &stubListProviderWebhooks{err: integrationsDomain.ErrProviderWebhookRequestFailed}
	h := newProviderWebhookTestHandler(stub, &stubCreateProviderWebhook{}, &stubDeleteProviderWebhook{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/provider-webhooks", nil)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestCreateProviderWebhookHandler_success(t *testing.T) {
	stub := &stubCreateProviderWebhook{out: &integrationsDomain.ProviderWebhook{ID: "1", Provider: integrationsDomain.ProviderWooCommerce, Topic: "order.updated", DeliveryURL: "https://commerceos.example.com/hook", Status: "active"}}
	h := newProviderWebhookTestHandler(&stubListProviderWebhooks{}, stub, &stubDeleteProviderWebhook{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/provider-webhooks", map[string]any{"topic": "order.updated"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "order.updated")
}

func TestCreateProviderWebhookHandler_noWebhookSecretSet_returns422(t *testing.T) {
	stub := &stubCreateProviderWebhook{err: integrationsDomain.ErrWebhookSecretNotSet}
	h := newProviderWebhookTestHandler(&stubListProviderWebhooks{}, stub, &stubDeleteProviderWebhook{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/provider-webhooks", map[string]any{"topic": "order.updated"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteProviderWebhookHandler_success(t *testing.T) {
	stub := &stubDeleteProviderWebhook{}
	h := newProviderWebhookTestHandler(&stubListProviderWebhooks{}, &stubCreateProviderWebhook{}, stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/provider-webhooks/42", nil)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteProviderWebhookHandler_liveCallFailure_returns502(t *testing.T) {
	stub := &stubDeleteProviderWebhook{err: integrationsDomain.ErrProviderWebhookRequestFailed}
	h := newProviderWebhookTestHandler(&stubListProviderWebhooks{}, &stubCreateProviderWebhook{}, stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/connections/conn-1/provider-webhooks/42", nil)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestSetWebhookSecretHandler_returnsPlaintextSecretOnce(t *testing.T) {
	stub := &stubSetWebhookSecret{out: &appintegrations.SetWebhookSecretResult{ConnectionID: "conn-1", PlaintextSecret: "topsecret"}}
	h := newWebhookAdminTestHandler(stub, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/webhook-secret", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "topsecret")
}

func TestSetWebhookSecretHandler_notFound(t *testing.T) {
	stub := &stubSetWebhookSecret{err: integrationsDomain.ErrConnectionNotFound}
	h := newWebhookAdminTestHandler(stub, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/missing/webhook-secret", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListWebhookFallbackEventsHandler_returnsList(t *testing.T) {
	stub := &stubListWebhookFallbackEvents{out: []webhooksDomain.FallbackEvent{{ID: "fb-1", Provider: "woocommerce", Status: webhooksDomain.FallbackStatusPendingRepublish}}}
	h := newWebhookAdminTestHandler(&stubSetWebhookSecret{}, stub, &stubRetryWebhookFallbackEvent{}, &stubAbandonWebhookFallbackEvent{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/webhook-fallback-events", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "fb-1")
}

func TestRetryWebhookFallbackEventHandler_success(t *testing.T) {
	stub := &stubRetryWebhookFallbackEvent{}
	h := newWebhookAdminTestHandler(&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, stub, &stubAbandonWebhookFallbackEvent{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/webhook-fallback-events/fb-1/retry", nil)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestRetryWebhookFallbackEventHandler_publishFailureReturns502(t *testing.T) {
	stub := &stubRetryWebhookFallbackEvent{err: errors.New("nats: no responders")}
	h := newWebhookAdminTestHandler(&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, stub, &stubAbandonWebhookFallbackEvent{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/webhook-fallback-events/fb-1/retry", nil)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}

func TestRetryWebhookFallbackEventHandler_notFound(t *testing.T) {
	stub := &stubRetryWebhookFallbackEvent{err: webhooksDomain.ErrFallbackEventNotFound}
	h := newWebhookAdminTestHandler(&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, stub, &stubAbandonWebhookFallbackEvent{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/webhook-fallback-events/missing/retry", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAbandonWebhookFallbackEventHandler_success(t *testing.T) {
	stub := &stubAbandonWebhookFallbackEvent{}
	h := newWebhookAdminTestHandler(&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/webhook-fallback-events/fb-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAbandonWebhookFallbackEventHandler_notFound(t *testing.T) {
	stub := &stubAbandonWebhookFallbackEvent{err: webhooksDomain.ErrFallbackEventNotFound}
	h := newWebhookAdminTestHandler(&stubSetWebhookSecret{}, &stubListWebhookFallbackEvents{}, &stubRetryWebhookFallbackEvent{}, stub)
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/webhook-fallback-events/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Bol handler tests ────────────────────────────────────────────────────────

func TestConnectBolHandler_success(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-bol-1", Provider: integrationsDomain.ProviderBol, Status: integrationsDomain.StatusConnected}
	h := newBolTestHandler(&stubConnectBol{out: conn}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/bol/connect", map[string]string{
		"client_id": "id-1", "client_secret": "secret-1", "economic_operator_id": "op-1",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "bol")
	assert.NotContains(t, w.Body.String(), "secret-1")
}

func TestConnectBolHandler_rejectsMissingFields(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/bol/connect", map[string]string{"client_id": "id-1"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPushProductToBolHandler_success(t *testing.T) {
	listings := []channelsDomain.ProductChannelListing{{ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "bol", ExternalID: "offer-1"}}
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{out: listings}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/push-product-bol", map[string]string{"product_id": "prod-1"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "offer-1")
}

func TestPushProductToBolHandler_missingEAN_returns422NotInternal(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{err: channelsDomain.ErrBolMissingEAN}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/push-product-bol", map[string]string{"product_id": "prod-1"})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
}

func TestPushProductToBolHandler_bulkPoolSKU_returns422NotInternal(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{err: channelsDomain.ErrCannotPushBulkPoolSKU}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/connections/conn-1/push-product-bol", map[string]string{"product_id": "prod-1"})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
}

func TestGetBolProductOverrideHandler_returnsNullWhenNoneExists(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{out: nil}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-overrides/prod-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":null`)
}

func TestUpsertBolProductOverrideHandler_success(t *testing.T) {
	override := &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1", ConditionCategory: channelsDomain.BolConditionNew, FulfilmentMethod: channelsDomain.BolFulfilmentFBR,
	}
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{out: override}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/connections/conn-1/bol-overrides/prod-1", map[string]any{
		"gpc_code": "80006362",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "NEW")
}

func TestUpsertBolProductOverrideHandler_passesCustomPriceThrough(t *testing.T) {
	customPrice := int64(2499)
	override := &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1", ConditionCategory: channelsDomain.BolConditionNew,
		FulfilmentMethod: channelsDomain.BolFulfilmentFBR, CustomPriceMinorUnits: &customPrice,
	}
	upsert := &stubUpsertBolProductOverride{out: override}
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, upsert, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/connections/conn-1/bol-overrides/prod-1", map[string]any{
		"custom_price_minor_units": 2499,
	})

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, upsert.gotInput.CustomPriceMinorUnits)
	assert.Equal(t, int64(2499), *upsert.gotInput.CustomPriceMinorUnits)
	assert.Contains(t, w.Body.String(), `"custom_price_minor_units":2499`)
}

func TestGetBolCommissionHandler_success(t *testing.T) {
	result := &appchannels.BolCommissionResult{EAN: "0630509892495", FixedAmount: 0.30, Percentage: 12.5}
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{out: result}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-commission/prod-1/sku-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "0630509892495")
	assert.Contains(t, w.Body.String(), "12.5")
}

func TestGetBolCommissionHandler_missingEAN_returns422NotInternal(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{err: channelsDomain.ErrBolMissingEAN}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-commission/prod-1/sku-1", nil)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
}

func TestGetBolCommissionHandler_skuNotFound_returns404NotInternal(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{err: pmsDomain.ErrSKUNotFound}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-commission/prod-1/sku-1", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "NOT_FOUND")
}

func TestGetBolCommissionHandler_bolAPIFailure_returns502NotInternal(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{err: channelsDomain.ErrBolCommissionRequestFailed}, &stubListBolEconomicOperators{})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-commission/prod-1/sku-1", nil)

	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "UPSTREAM_ERROR")
}

func TestListBolEconomicOperatorsHandler_success(t *testing.T) {
	operators := []integrationsDomain.BolEconomicOperator{
		{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: "Cantinho Brasileiro", Status: "VALID"},
	}
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{out: operators})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-economic-operators", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "3fa85f64-5717-4562-b3fc-2c963f66afa6")
	assert.Contains(t, w.Body.String(), "Cantinho Brasileiro")
}

func TestListBolEconomicOperatorsHandler_bolAPIFailure_returns502NotInternal(t *testing.T) {
	h := newBolTestHandler(&stubConnectBol{}, &stubPushProductToBol{}, &stubGetBolProductOverride{}, &stubUpsertBolProductOverride{}, &stubGetBolProductContent{}, &stubUpsertBolProductContent{}, &stubGetBolCommission{}, &stubListBolEconomicOperators{err: integrationsDomain.ErrBolEconomicOperatorsRequestFailed})
	r := gin.New()
	integrationshandler.RegisterRoutes(r.Group("/api/v1"), h)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/connections/conn-1/bol-economic-operators", nil)

	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "UPSTREAM_ERROR")
}
