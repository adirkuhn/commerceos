package channels

import (
	"context"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	inventoryDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// warehouseExistenceChecker is inventory.WarehouseRepository narrowed to
// point of use (ISP) -- Create/UpdateSalesChannelUseCase only ever need to
// confirm a given DefaultWarehouseID is real before saving it.
type warehouseExistenceChecker interface {
	FindByID(ctx context.Context, id string) (*inventoryDomain.Warehouse, error)
}

// companySettingsReader is settings.Repository narrowed to point of use: a
// new channel's storefront speaks the company language unless told
// otherwise.
type companySettingsReader interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type CreateSalesChannelInput struct {
	Name                       string
	Type                       channelsDomain.ChannelType
	ConnectionID               *string
	EnrollmentMode             channelsDomain.EnrollmentMode
	EnrollmentCategoryID       *string
	EnrollmentTag              string
	PriceAdjustmentBasisPoints *int
	QuantityBreaks             []channelsDomain.QuantityBreak
	// CurrencyCode, "" meaning "use the tenant's default currency", is
	// only settable here -- see SalesChannel.CurrencyCode's own doc
	// comment for why it's immutable after creation.
	CurrencyCode string
	// DefaultWarehouseID, nil meaning unset, is required (checked at
	// checkout/order-placement time, not here) before a Type without its
	// own Connection (Internal, WhatsApp) can actually process an order --
	// see SalesChannel.DefaultWarehouseID's own doc comment. Unlike
	// CurrencyCode, changeable later via UpdateSalesChannelUseCase.
	DefaultWarehouseID *string
	// StorefrontMode, "" meaning headless -- see channels.StorefrontMode.
	StorefrontMode channelsDomain.StorefrontMode
	// SignInURL, nil or blank meaning none -- see channels.ResolveSignInURL.
	SignInURL *string
	// StorefrontLocale, "" meaning the company language (settings.TenantSettings.Language).
	StorefrontLocale channelsDomain.StorefrontLocale
	// StorefrontLocales, the other languages shoppers can switch to (the
	// default is always offered) -- see channels.ResolveStorefrontLocales.
	StorefrontLocales []channelsDomain.StorefrontLocale
}

// validateSalesChannelInput is shared by Create and Update -- same field
// rules apply whichever way a channel got into this shape.
func validateSalesChannelInput(channelType channelsDomain.ChannelType, connectionID *string, mode channelsDomain.EnrollmentMode, categoryID *string, tag string, priceAdjustmentBasisPoints *int, quantityBreaks []channelsDomain.QuantityBreak) error {
	if !channelType.Valid() {
		return channelsDomain.ErrInvalidChannelType
	}
	if !mode.Valid() {
		return channelsDomain.ErrInvalidEnrollmentMode
	}
	if channelType.RequiresConnection() && (connectionID == nil || *connectionID == "") {
		return channelsDomain.ErrConnectionRequired
	}
	if mode == channelsDomain.EnrollmentModeCategory && (categoryID == nil || *categoryID == "") {
		return channelsDomain.ErrEnrollmentCategoryRequired
	}
	if mode == channelsDomain.EnrollmentModeTag && tag == "" {
		return channelsDomain.ErrEnrollmentTagRequired
	}
	if priceAdjustmentBasisPoints != nil && (*priceAdjustmentBasisPoints < 0 || *priceAdjustmentBasisPoints > 10000) {
		return channelsDomain.ErrInvalidPriceAdjustment
	}
	seenMinQuantity := map[int64]bool{}
	for _, b := range quantityBreaks {
		if b.MinQuantity <= 0 || b.DiscountBasisPoints < 0 || b.DiscountBasisPoints > 10000 || seenMinQuantity[b.MinQuantity] {
			return channelsDomain.ErrInvalidQuantityBreak
		}
		seenMinQuantity[b.MinQuantity] = true
	}
	return nil
}

type CreateSalesChannelUseCase struct {
	db            *gorm.DB
	repo          channelsDomain.SalesChannelRepository
	connRepo      integrationsDomain.ConnectionRepository
	fxRepo        fxDomain.ExchangeRateRepository
	warehouseRepo warehouseExistenceChecker
	company       companySettingsReader
}

func NewCreateSalesChannelUseCase(
	db *gorm.DB, repo channelsDomain.SalesChannelRepository, connRepo integrationsDomain.ConnectionRepository,
	fxRepo fxDomain.ExchangeRateRepository, warehouseRepo warehouseExistenceChecker, company companySettingsReader,
) *CreateSalesChannelUseCase {
	return &CreateSalesChannelUseCase{db: db, repo: repo, connRepo: connRepo, fxRepo: fxRepo, warehouseRepo: warehouseRepo, company: company}
}

func (uc *CreateSalesChannelUseCase) Execute(ctx context.Context, in CreateSalesChannelInput) (*channelsDomain.SalesChannel, error) {
	if err := validateSalesChannelInput(in.Type, in.ConnectionID, in.EnrollmentMode, in.EnrollmentCategoryID, in.EnrollmentTag, in.PriceAdjustmentBasisPoints, in.QuantityBreaks); err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}
	if in.CurrencyCode != "" && !fxDomain.ValidCurrencyCode(in.CurrencyCode) {
		return nil, fmt.Errorf("createSalesChannel: %w", channelsDomain.ErrInvalidCurrencyCode)
	}
	storefrontMode, err := channelsDomain.ResolveStorefrontMode(in.Type, in.StorefrontMode)
	if err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}
	storefrontLocale, err := channelsDomain.ResolveStorefrontLocale(in.StorefrontLocale)
	if err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}
	if err := channelsDomain.ValidateStorefrontLocales(in.StorefrontLocales); err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}
	var signInURL *string
	if in.SignInURL != nil {
		if signInURL, err = channelsDomain.ResolveSignInURL(in.Type, *in.SignInURL); err != nil {
			return nil, fmt.Errorf("createSalesChannel: %w", err)
		}
	}

	var channel channelsDomain.SalesChannel
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if in.ConnectionID != nil {
			conn, err := uc.connRepo.FindByID(ctx, *in.ConnectionID)
			if err != nil {
				return err
			}
			if string(conn.Provider) != string(in.Type) {
				return channelsDomain.ErrChannelTypeConnectionMismatch
			}
		}
		if in.CurrencyCode != "" {
			rate, err := uc.fxRepo.FindByCurrency(ctx, in.CurrencyCode)
			if err != nil {
				return err
			}
			if rate == nil {
				return channelsDomain.ErrChannelCurrencyRequiresRate
			}
		}
		if in.DefaultWarehouseID != nil {
			if _, err := uc.warehouseRepo.FindByID(ctx, *in.DefaultWarehouseID); err != nil {
				return err
			}
		}
		if in.StorefrontLocale == "" {
			settings, err := uc.company.Get(ctx)
			if err != nil {
				return err
			}
			storefrontLocale = settings.Language.OrDefault()
		}
		storefrontLocales, err := channelsDomain.ResolveStorefrontLocales(storefrontLocale, in.StorefrontLocales)
		if err != nil {
			return err
		}
		now := time.Now()
		channel = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: in.Name, Type: in.Type, ConnectionID: in.ConnectionID,
			EnrollmentMode: in.EnrollmentMode, EnrollmentCategoryID: in.EnrollmentCategoryID, EnrollmentTag: in.EnrollmentTag,
			PriceAdjustmentBasisPoints: in.PriceAdjustmentBasisPoints,
			QuantityBreaks:             in.QuantityBreaks,
			CurrencyCode:               in.CurrencyCode,
			DefaultWarehouseID:         in.DefaultWarehouseID,
			StorefrontMode:             storefrontMode,
			SignInURL:                  signInURL,
			StorefrontLocale:           storefrontLocale,
			StorefrontLocales:          storefrontLocales,
			Active:                     true, CreatedAt: now, UpdatedAt: now,
		}
		return uc.repo.Create(ctx, &channel)
	})
	if err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}
	return &channel, nil
}
