package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
)

func TestGetBolProductContent_returnsNilWhenNoneExists(t *testing.T) {
	repo := &fakeBolContentRepo{}
	uc := appchannels.NewGetBolProductContentUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestUpsertBolProductContent_savesFields(t *testing.T) {
	repo := &fakeBolContentRepo{}
	uc := appchannels.NewUpsertBolProductContentUseCase(newMockGormDB(t, 1), repo)
	brand := "Hipoglós"
	widthMM := int32(80)

	result, err := uc.Execute(context.Background(), appchannels.UpsertBolProductContentInput{
		ProductID: "prod-1", ConnectionID: "conn-1", Brand: &brand, PackageWidthMM: &widthMM,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Brand)
	assert.Equal(t, "Hipoglós", *result.Brand)
	require.Len(t, repo.upserted, 1)
	require.NotNil(t, repo.upserted[0].PackageWidthMM)
	assert.Equal(t, int32(80), *repo.upserted[0].PackageWidthMM)
	assert.Equal(t, "prod-1", repo.upserted[0].ProductID)
}
