package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestCreateSalesChannel_woocommerce_success(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderWooCommerce}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

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
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

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
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, connRepo, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "BOL", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrChannelTypeConnectionMismatch)
}

func TestCreateSalesChannel_rejectsInvalidType(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelType("shopify"), EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidChannelType)
}

func TestCreateSalesChannel_rejectsInvalidEnrollmentMode(t *testing.T) {
	connID := "conn-1"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentMode("random"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidEnrollmentMode)
}

func TestCreateSalesChannel_requiresConnectionForWooCommerce(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrConnectionRequired)
}

func TestCreateSalesChannel_requiresCategoryForCategoryMode(t *testing.T) {
	connID := "conn-1"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeCategory,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrEnrollmentCategoryRequired)
}

func TestCreateSalesChannel_requiresTagForTagMode(t *testing.T) {
	connID := "conn-1"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeTag,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrEnrollmentTagRequired)
}

func TestCreateSalesChannel_internal_withPriceAdjustment_success(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	discount := 2000
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		PriceAdjustmentBasisPoints: &discount,
	})

	require.NoError(t, err)
	require.NotNil(t, channel.PriceAdjustmentBasisPoints)
	assert.Equal(t, 2000, *channel.PriceAdjustmentBasisPoints)
}

func TestCreateSalesChannel_rejectsPriceAdjustmentOutOfRange(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})
	tooHigh := 10001

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		PriceAdjustmentBasisPoints: &tooHigh,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidPriceAdjustment)
}

func TestCreateSalesChannel_rejectsNegativePriceAdjustment(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})
	negative := -1

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		PriceAdjustmentBasisPoints: &negative,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidPriceAdjustment)
}

func TestCreateSalesChannel_savesQuantityBreaks(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

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
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: 0, DiscountBasisPoints: 1500}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidQuantityBreak)
}

func TestCreateSalesChannel_rejectsQuantityBreakDiscountOutOfRange(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		QuantityBreaks: []channelsDomain.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 10001}},
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidQuantityBreak)
}

func TestCreateSalesChannel_rejectsDuplicateMinQuantityAcrossBreaks(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

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
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, connRepo, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "X", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}

func TestCreateSalesChannel_rejectsInvalidCurrencyCode(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 0), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		CurrencyCode: "usd",
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidCurrencyCode)
}

func TestCreateSalesChannel_rejectsNonDefaultCurrencyWithNoExchangeRate(t *testing.T) {
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2B Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		CurrencyCode: "USD",
	})

	assert.ErrorIs(t, err, channelsDomain.ErrChannelCurrencyRequiresRate)
}

func TestCreateSalesChannel_acceptsNonDefaultCurrencyWithAnExchangeRate(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	fxRepo := &stubExchangeRateRepo{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 10876}}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, fxRepo, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "US Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		CurrencyCode: "USD",
	})

	require.NoError(t, err)
	assert.Equal(t, "USD", channel.CurrencyCode)
}

func TestCreateSalesChannel_emptyCurrencyCodeNeedsNoExchangeRate(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Default Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	require.NoError(t, err)
	assert.Equal(t, "", channel.CurrencyCode)
}

func TestCreateSalesChannel_acceptsARealDefaultWarehouse(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	whID := "wh-1"
	warehouseRepo := &stubWarehouseExistenceChecker{byID: map[string]invDomain.Warehouse{whID: {ID: whID, Name: "Main DC"}}}
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{}, &stubExchangeRateRepo{}, warehouseRepo, &stubTenantSettings{})

	channel, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2C Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		DefaultWarehouseID: &whID,
	})

	require.NoError(t, err)
	require.NotNil(t, channel.DefaultWarehouseID)
	assert.Equal(t, whID, *channel.DefaultWarehouseID)
}

func TestCreateSalesChannel_rejectsAnUnknownDefaultWarehouse(t *testing.T) {
	whID := "missing-wh"
	uc := appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, 1), &stubSalesChannelRepo{}, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})

	_, err := uc.Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "B2C Store", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		DefaultWarehouseID: &whID,
	})

	assert.ErrorIs(t, err, invDomain.ErrWarehouseNotFound)
}
