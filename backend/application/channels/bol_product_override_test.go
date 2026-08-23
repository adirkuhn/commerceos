package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
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
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo)

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

func TestUpsertBolProductOverride_defaultsConditionAndFulfilmentWhenEmpty(t *testing.T) {
	repo := &fakeBolOverrideRepo{}
	uc := appchannels.NewUpsertBolProductOverrideUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductOverrideInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.BolConditionNew, result.ConditionCategory)
	assert.Equal(t, channelsDomain.BolFulfilmentFBR, result.FulfilmentMethod)
}
