package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// StoreInfo is what a storefront shows about itself.
type StoreInfo struct {
	Name         string
	CurrencyCode string
	Locale       channelsDomain.StorefrontLocale
	// Locales is every language shoppers can switch to, Locale first.
	Locales []channelsDomain.StorefrontLocale
}

type tenantSettingsGetter interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type GetStoreInfoUseCase struct {
	db       *gorm.DB
	channels channelsDomain.SalesChannelRepository
	settings tenantSettingsGetter
}

func NewGetStoreInfoUseCase(db *gorm.DB, channels channelsDomain.SalesChannelRepository, settings tenantSettingsGetter) *GetStoreInfoUseCase {
	return &GetStoreInfoUseCase{db: db, channels: channels, settings: settings}
}

func (uc *GetStoreInfoUseCase) Execute(ctx context.Context, salesChannelID string) (*StoreInfo, error) {
	var out StoreInfo
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		ch, err := findInternalChannel(ctx, uc.channels, salesChannelID)
		if err != nil {
			return err
		}
		locale, err := channelsDomain.ResolveStorefrontLocale(ch.StorefrontLocale)
		if err != nil {
			return err
		}
		out = StoreInfo{Name: ch.Name, CurrencyCode: ch.CurrencyCode, Locale: locale, Locales: ch.OfferedStorefrontLocales()}
		if out.CurrencyCode == "" {
			s, err := uc.settings.Get(ctx)
			if err != nil {
				return err
			}
			if s != nil {
				out.CurrencyCode = s.DefaultCurrency
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getStoreInfo: %w", err)
	}
	return &out, nil
}
