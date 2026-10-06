package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateSalesChannel_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	connID := "conn-1"
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, connRepo, &stubExchangeRateRepo{},
		&stubWarehouseExistenceChecker{}, &stubTenantSettings{}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "My Store", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sales_channel.created", e.Subject)
	assert.Equal(t, "channel-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"name": "My Store", "type": "woocommerce", "connection_id": "conn-1", "enrollment_mode": "all", "currency_code": "",
	}, e.Data())
}

func TestUpdateSalesChannel_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{
		ID: "channel-1", Name: "Old", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: strPtr("conn-1"),
		EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
	}}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "New", ConnectionID: strPtr("conn-1"), EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: false,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sales_channel.updated", e.Subject)
	changes := e.Changes()
	assert.Equal(t, [2]any{"Old", "New"}, changes["name"])
	assert.Equal(t, [2]any{true, false}, changes["active"])
	assert.NotContains(t, changes, "connection_id")
}

func TestDeleteSalesChannel_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1", Name: "Shop", Type: channelsDomain.ChannelTypeBol}}
	uc := appchannels.NewDeleteSalesChannelUseCase(newMockGormDB(t, 1), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "channel-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sales_channel.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Shop", "type": "bol"}, e.Data())
}

func TestSetSalesChannelPaymentMethod_RecordsChange(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "pay-1", Provider: integrationsDomain.ProviderStripe, Status: integrationsDomain.StatusConnected}}
	uc := appchannels.NewSetSalesChannelPaymentMethodUseCase(newMockGormDB(t, 1), repo, connRepo, outbox)

	_, err := uc.Execute(context.Background(), "channel-1", strPtr("pay-1"))

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sales_channel.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"payment_connection_id": {nil, "pay-1"}}, e.Changes())
}

func TestSetProductChannelOverride_RecordsSet(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewSetProductChannelOverrideUseCase(newMockGormDB(t, 1), &stubSalesChannelProductOverrideRepo{},
		&stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}},
		&stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"p-1": {ID: "p-1"}}}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.SetProductChannelOverrideInput{ChannelID: "channel-1", ProductID: "p-1", Included: false})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sales_channel_product_override.set", e.Subject)
	assert.Equal(t, "override-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"channel_id": "channel-1", "product_id": "p-1", "included": false}, e.Data())
}

func TestUpsertExternalCategoryLink_RecordsSaved(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewUpsertExternalCategoryLinkUseCase(newMockGormDB(t, 1), &stubExternalCategoryLinkRepo{},
		&stubResolveCategoryRepo{byID: map[string]pmsDomain.Category{"cat-1": {ID: "cat-1"}}}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.UpsertExternalCategoryLinkInput{ConnectionID: "conn-1", ExternalID: "42", CategoryID: "cat-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.external_category_link.saved", e.Subject)
	assert.Equal(t, "link-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"connection_id": "conn-1", "provider": "woocommerce", "external_id": "42", "category_id": "cat-1", "source": "manual",
	}, e.Data())
}

func TestDeleteExternalCategoryLink_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewDeleteExternalCategoryLinkUseCase(newMockGormDB(t, 1), &stubExternalCategoryLinkRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "link-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.external_category_link.deleted", e.Subject)
	assert.Equal(t, "link-1", e.Payload["entity_id"])
}

func TestUpsertSyncSchedule_RecordsSaved(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), &stubSyncScheduleRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories, Direction: channelsDomain.SyncDirectionImport, IntervalMinutes: 60,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.sync_schedule.saved", e.Subject)
	assert.Equal(t, map[string]any{
		"connection_id": "conn-1", "entity_type": "categories", "direction": "import", "enabled": false, "interval_minutes": float64(60),
		"default_warehouse_id": nil, "default_currency": nil,
	}, e.Data())
}

func TestUpsertSyncSchedules_RecordsEachSchedule(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewUpsertSyncSchedulesUseCase(newMockGormDB(t, 1), &stubSyncScheduleRepo{}, outbox)
	one := appchannels.UpsertSyncScheduleInput{ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories, Direction: channelsDomain.SyncDirectionImport, IntervalMinutes: 60}
	two := one
	two.Direction = channelsDomain.SyncDirectionExport

	_, err := uc.Execute(context.Background(), appchannels.UpsertSyncSchedulesInput{Schedules: []appchannels.UpsertSyncScheduleInput{one, two}})

	require.NoError(t, err)
	assert.Equal(t, []string{"channels.sync_schedule.saved", "channels.sync_schedule.saved"}, outbox.Subjects())
}

func TestCreateImportJob_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewCreateImportJobUseCase(newMockGormDB(t, 1), &stubImportJobRepo{}, &stubListingRepo{}, &stubNotifier{}, outbox)

	_, err := uc.Execute(context.Background(), appchannels.CreateImportJobInput{ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.import_job.created", e.Subject)
	assert.Equal(t, "job-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"connection_id": "conn-1", "default_warehouse_id": "wh-1", "default_currency": "EUR", "is_first_import": true}, e.Data())
}

func TestConfigUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()

	err := appchannels.NewDeleteSalesChannelUseCase(newMockGormDB(t, 1),
		&stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}}, failing).Execute(ctx, "channel-1")
	assert.ErrorIs(t, err, boom, "delete channel")
	assert.ErrorIs(t, appchannels.NewDeleteExternalCategoryLinkUseCase(newMockGormDB(t, 1), &stubExternalCategoryLinkRepo{}, failing).Execute(ctx, "link-1"), boom, "delete link")
	_, err = appchannels.NewCreateImportJobUseCase(newMockGormDB(t, 1), &stubImportJobRepo{}, &stubListingRepo{}, &stubNotifier{}, failing).
		Execute(ctx, appchannels.CreateImportJobInput{ConnectionID: "conn-1"})
	assert.ErrorIs(t, err, boom, "import job")
	_, err = appchannels.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), &stubSyncScheduleRepo{}, failing).Execute(ctx, appchannels.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories, Direction: channelsDomain.SyncDirectionImport, IntervalMinutes: 60,
	})
	assert.ErrorIs(t, err, boom, "schedule")
}
