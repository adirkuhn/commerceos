package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestCreateSalesChannel_woocommerce_success(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo)

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "My Store", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "My Store", channel.Name)
	assert.True(t, channel.Active)
}

func TestCreateSalesChannel_rejectsInvalidType(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelType("shopify"), EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidChannelType)
}

func TestCreateSalesChannel_rejectsInvalidEnrollmentMode(t *testing.T) {
	connID := "conn-1"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentMode("random"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidEnrollmentMode)
}

func TestCreateSalesChannel_requiresConnectionForWooCommerce(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrConnectionRequired)
}

func TestCreateSalesChannel_requiresCategoryForCategoryMode(t *testing.T) {
	connID := "conn-1"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeCategory,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrEnrollmentCategoryRequired)
}

func TestCreateSalesChannel_requiresTagForTagMode(t *testing.T) {
	connID := "conn-1"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeTag,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrEnrollmentTagRequired)
}

func TestCreateSalesChannel_propagatesConnectionNotFound(t *testing.T) {
	connID := "missing"
	connRepo := &stubConnectionRepo{findErr: integrationsDomain.ErrConnectionNotFound}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, connRepo)

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
