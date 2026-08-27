package channels_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
)

func TestGetBolProductOverride_returnsNilWhenNoneExists(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	uc := appchannels.NewGetBolProductOverrideUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestUpsertBolProductOverride_savesFields(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo, &fakeOutboxRepo{})

	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{
		ProductID: "prod-1", ConnectionID: "conn-1",
		EANOverride: strPtr("9999999999999"), GPCCode: strPtr("80006362"),
		ConditionCategory: "REFURBISHED", FulfilmentMethod: "FBR",
	})

	require.NoError(t, err)
	require.Len(t, repo.upserted, 1)
	assert.Equal(t, "prod-1", repo.upserted[0].ProductID)
	assert.Equal(t, channelsDomain.ConditionCategory("REFURBISHED"), result.ConditionCategory)
}

func TestUpsertBolProductOverride_savesCustomPrice(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo, &fakeOutboxRepo{})

	price := int64(2499)
	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{
		ProductID: "prod-1", ConnectionID: "conn-1", CustomPriceMinorUnits: &price,
	})

	require.NoError(t, err)
	require.NotNil(t, result.CustomPriceMinorUnits)
	assert.Equal(t, int64(2499), *result.CustomPriceMinorUnits)
	require.Len(t, repo.upserted, 1)
	require.NotNil(t, repo.upserted[0].CustomPriceMinorUnits)
	assert.Equal(t, int64(2499), *repo.upserted[0].CustomPriceMinorUnits)
}

func TestUpsertBolProductOverride_savesNameOverride(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo, &fakeOutboxRepo{})

	name := "Custom Bol Title"
	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{
		ProductID: "prod-1", ConnectionID: "conn-1", NameOverride: &name,
	})

	require.NoError(t, err)
	require.NotNil(t, result.NameOverride)
	assert.Equal(t, "Custom Bol Title", *result.NameOverride)
	require.Len(t, repo.upserted, 1)
	require.NotNil(t, repo.upserted[0].NameOverride)
	assert.Equal(t, "Custom Bol Title", *repo.upserted[0].NameOverride)
}

// TestUpsertBolProductOverride_raisesProductChangedEvent proves saving the
// Bol tab's Custom price (or any other override field) raises the same
// pms.product.changed event a normal product edit does -- SyncProductToChannelsUseCase
// is the only thing that actually pushes the new price to Bol's offer
// (PushProductToBolUseCase reads CustomPriceMinorUnits), so without this
// enqueue the override saves to the DB but Bol never hears about it. Mirrors
// UpsertBolProductContentUseCase's equivalent test.
func TestUpsertBolProductOverride_raisesProductChangedEvent(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	outboxRepo := &fakeOutboxRepo{}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo, outboxRepo)

	_, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{
		ProductID: "prod-1", ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, outboxRepo.enqueued, 1)
	row := outboxRepo.enqueued[0]
	assert.Equal(t, "pms.product.changed", row.Subject)
	assert.NotEmpty(t, row.EventID)

	var event events.ProductChanged
	require.NoError(t, json.Unmarshal(row.Payload, &event))
	assert.Equal(t, "prod-1", event.ProductID)
	assert.Equal(t, row.EventID, event.EventID)
}

func TestUpsertBolProductOverride_propagatesOutboxEnqueueFailure(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	outboxRepo := &fakeOutboxRepo{err: assert.AnError}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo, outboxRepo)

	_, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.Error(t, err, "the override write must not silently succeed while the event fails to enqueue -- both happen in the same transaction")
}

func TestUpsertBolProductOverride_defaultsConditionAndFulfilmentWhenEmpty(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo, &fakeOutboxRepo{})

	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.BolConditionNew, result.ConditionCategory)
	assert.Equal(t, channelsDomain.BolFulfilmentFBR, result.FulfilmentMethod)
}
