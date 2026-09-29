package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

type slugProductRepo struct {
	fakeProductRepo
	bySlug map[string]pmsDomain.Product
}

func (f *slugProductRepo) FindBySlug(_ context.Context, slug string) (*pmsDomain.Product, error) {
	if p, ok := f.bySlug[slug]; ok {
		return &p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}

type stubGetProduct struct{ gotChannel, gotID string }

func (s *stubGetProduct) Execute(_ context.Context, channel, id string) (*appstorefront.ProductDetail, error) {
	s.gotChannel, s.gotID = channel, id
	return &appstorefront.ProductDetail{}, nil
}

func TestGetProductBySlug_delegatesWithTheID(t *testing.T) {
	repo := &slugProductRepo{bySlug: map[string]pmsDomain.Product{"red-shoes": {ID: "p1"}}}
	get := &stubGetProduct{}
	uc := appstorefront.NewGetProductBySlugUseCase(newMockGormDB(t, 2), repo, get)

	_, err := uc.Execute(context.Background(), "channel-1", "red-shoes")
	require.NoError(t, err)
	assert.Equal(t, "p1", get.gotID)
	assert.Equal(t, "channel-1", get.gotChannel)

	_, err = uc.Execute(context.Background(), "channel-1", "nope")
	assert.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}

type stubSettingsRepo struct {
	s *settingsDomain.TenantSettings
}

func (s stubSettingsRepo) Get(context.Context) (*settingsDomain.TenantSettings, error) {
	return s.s, nil
}

func TestGetStoreInfo(t *testing.T) {
	ch := internalChannel("channel-1")
	ch.Name = "My Shop"
	channels := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": ch}}
	uc := appstorefront.NewGetStoreInfoUseCase(newMockGormDB(t, 2), channels, stubSettingsRepo{s: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}})

	info, err := uc.Execute(context.Background(), "channel-1")
	require.NoError(t, err)
	assert.Equal(t, appstorefront.StoreInfo{Name: "My Shop", CurrencyCode: "EUR", Locale: channelsDomain.StorefrontLocaleEnglish}, *info, "no language set reads as English")

	ch.CurrencyCode = "GBP"
	ch.StorefrontLocale = channelsDomain.StorefrontLocalePortuguese
	channels.byID["channel-1"] = ch
	info, err = uc.Execute(context.Background(), "channel-1")
	require.NoError(t, err)
	assert.Equal(t, "GBP", info.CurrencyCode, "a channel's own currency wins")
	assert.Equal(t, channelsDomain.StorefrontLocalePortuguese, info.Locale)
}
