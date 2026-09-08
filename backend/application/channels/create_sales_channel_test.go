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
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
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

func TestCreateSalesChannel_bol_success(t *testing.T) {
	connID := "conn-bol-1"
	repo := &stubSalesChannelRepo{}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderBol}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo)

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "BOL", Type: channelsDomain.ChannelTypeBol, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeManual,
	})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, channelsDomain.ChannelTypeBol, channel.Type)
}

// TestCreateSalesChannel_rejectsTypeConnectionMismatch is a regression test
// for a real prod bug: the create form used to hardcode Type=woocommerce
// regardless of which connection was picked, so a channel named "BOL"
// pointed at a real Bol connection but saved Type=woocommerce -- silently
// invisible to every ch.Type==ChannelTypeBol check (enrollment, scheduled
// export) forever, with no error anywhere to surface it. This guards the
// mismatch server-side regardless of what any future frontend/caller sends.
func TestCreateSalesChannel_rejectsTypeConnectionMismatch(t *testing.T) {
	connID := "conn-bol-1"
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderBol}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, connRepo)

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "BOL", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrChannelTypeConnectionMismatch)
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

func TestCreateSalesChannel_internal_withPriceAdjustment_success(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	discount := 2000
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{})

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		PriceAdjustmentBasisPoints: &discount,
	})

	require.NoError(t, err)
	require.NotNil(t, channel.PriceAdjustmentBasisPoints)
	assert.Equal(t, 2000, *channel.PriceAdjustmentBasisPoints)
}

func TestCreateSalesChannel_rejectsPriceAdjustmentOutOfRange(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})
	tooHigh := 10001

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		PriceAdjustmentBasisPoints: &tooHigh,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidPriceAdjustment)
}

func TestCreateSalesChannel_rejectsNegativePriceAdjustment(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})
	negative := -1

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		PriceAdjustmentBasisPoints: &negative,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidPriceAdjustment)
}

func TestCreateSalesChannel_savesQuantityBreaks(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{})

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 1500}},
	})

	require.NoError(t, err)
	require.Len(t, channel.QuantityBreaks, 1)
	assert.Equal(t, int64(10), channel.QuantityBreaks[0].MinQuantity)
	assert.Equal(t, 1500, channel.QuantityBreaks[0].DiscountBasisPoints)
}

func TestCreateSalesChannel_rejectsQuantityBreakWithZeroMinQuantity(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: 0, DiscountBasisPoints: 1500}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidQuantityBreak)
}

func TestCreateSalesChannel_rejectsQuantityBreakDiscountOutOfRange(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 10001}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidQuantityBreak)
}

func TestCreateSalesChannel_rejectsDuplicateMinQuantityAcrossBreaks(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		QuantityBreaks: []channelsDomain.QuantityBreak{
			{MinQuantity: 10, DiscountBasisPoints: 1000},
			{MinQuantity: 10, DiscountBasisPoints: 2000},
		},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidQuantityBreak)
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
