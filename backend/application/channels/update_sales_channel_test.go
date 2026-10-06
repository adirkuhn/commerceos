package channels_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func existingWooChannel() *channelsDomain.SalesChannel {
	connID := "conn-1"
	return &channelsDomain.SalesChannel{
		ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true,
	}
}

func TestUpdateSalesChannel_changesToAllMode_isTheAddAllBulkAction(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.EnrollmentModeAll, channel.EnrollmentMode)
	require.Len(t, repo.updated, 1)
}

func TestUpdateSalesChannel_changesToTagMode_isTheAddByTagBulkAction(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeTag, EnrollmentTag: "summer", Active: true,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.EnrollmentModeTag, channel.EnrollmentMode)
	assert.Equal(t, "summer", channel.EnrollmentTag)
}

func TestUpdateSalesChannel_propagatesNotFound(t *testing.T) {
	repo := &stubSalesChannelRepo{findErr: channelsDomain.ErrSalesChannelNotFound}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{ID: "missing", EnrollmentMode: channelsDomain.EnrollmentModeAll})

	assert.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}

func TestUpdateSalesChannel_rejectsInvalidEnrollmentModeOnExistingChannel(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", ConnectionID: strPtr("conn-1"), EnrollmentMode: channelsDomain.EnrollmentMode("bogus"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidEnrollmentMode)
}

// TestUpdateSalesChannel_rejectsRetargetingToAMismatchedProviderConnection
// guards the edit path against the same bug the create path had: Type is
// immutable via update (see UpdateSalesChannelInput's fields -- no Type),
// so re-targeting ConnectionID to a connection of a different provider
// would otherwise silently produce the exact same Type/provider mismatch.
func TestUpdateSalesChannel_rejectsRetargetingToAMismatchedProviderConnection(t *testing.T) {
	bolConnID := "conn-bol-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: bolConnID, Provider: integrationsDomain.ProviderBol}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &bolConnID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrChannelTypeConnectionMismatch)
}

func TestUpdateSalesChannel_canDeactivate(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: false,
	})

	require.NoError(t, err)
	assert.False(t, channel.Active)
}

func TestUpdateSalesChannel_replacesQuantityBreaksWholesale(t *testing.T) {
	connID := "conn-1"
	existing := existingWooChannel()
	existing.QuantityBreaks = []channelsDomain.QuantityBreak{{MinQuantity: 5, DiscountBasisPoints: 500}}
	repo := &stubSalesChannelRepo{findResult: existing}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 1500}},
	})

	require.NoError(t, err)
	require.Len(t, channel.QuantityBreaks, 1)
	assert.Equal(t, int64(10), channel.QuantityBreaks[0].MinQuantity)
}

func TestUpdateSalesChannel_rejectsInvalidQuantityBreak(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: -1, DiscountBasisPoints: 1500}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidQuantityBreak)
}

func TestUpdateSalesChannel_setsARealDefaultWarehouse(t *testing.T) {
	connID := "conn-1"
	whID := "wh-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	warehouseRepo := &stubWarehouseExistenceChecker{byID: map[string]invDomain.Warehouse{whID: {ID: whID, Name: "Main DC"}}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, warehouseRepo, &eventrecordtest.Outbox{})

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true,
		DefaultWarehouseID: &whID,
	})

	require.NoError(t, err)
	require.NotNil(t, channel.DefaultWarehouseID)
	assert.Equal(t, whID, *channel.DefaultWarehouseID)
}

func TestUpdateSalesChannel_rejectsAnUnknownDefaultWarehouse(t *testing.T) {
	connID := "conn-1"
	whID := "missing-wh"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubWarehouseExistenceChecker{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true,
		DefaultWarehouseID: &whID,
	})

	assert.ErrorIs(t, err, invDomain.ErrWarehouseNotFound)
}
