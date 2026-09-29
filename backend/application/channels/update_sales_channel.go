package channels

import (
	"context"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateSalesChannelInput struct {
	ID   string
	Name string
	// ConnectionID re-targets which integrations.Connection this channel
	// pushes through -- required when the channel's Type needs one (every
	// type implemented today).
	ConnectionID *string
	// EnrollmentMode/EnrollmentCategoryID/EnrollmentTag are how the "add
	// all"/"add by category"/"add by tag" bulk actions work: they're just
	// this channel's default rule changing, evaluated live against the
	// whole catalog with no per-product rows to write.
	EnrollmentMode             channelsDomain.EnrollmentMode
	EnrollmentCategoryID       *string
	EnrollmentTag              string
	PriceAdjustmentBasisPoints *int
	QuantityBreaks             []channelsDomain.QuantityBreak
	// DefaultWarehouseID, unlike CurrencyCode, is changeable after
	// creation -- see SalesChannel.DefaultWarehouseID's own doc comment.
	DefaultWarehouseID *string
	// StorefrontMode, "" meaning "keep the current mode" -- see
	// channels.StorefrontMode.
	StorefrontMode channelsDomain.StorefrontMode
	// SignInURL: nil keeps the current page, blank clears it -- see
	// channels.ResolveSignInURL.
	SignInURL *string
	// StorefrontLocale, "" meaning "keep the current language".
	StorefrontLocale channelsDomain.StorefrontLocale
	// StorefrontLocales, nil meaning "keep", replaces the other languages
	// shoppers can switch to -- see channels.ResolveStorefrontLocales.
	StorefrontLocales *[]channelsDomain.StorefrontLocale
	Active            bool
}

type UpdateSalesChannelUseCase struct {
	db            *gorm.DB
	repo          channelsDomain.SalesChannelRepository
	connRepo      integrationsDomain.ConnectionRepository
	warehouseRepo warehouseExistenceChecker
}

func NewUpdateSalesChannelUseCase(
	db *gorm.DB, repo channelsDomain.SalesChannelRepository, connRepo integrationsDomain.ConnectionRepository,
	warehouseRepo warehouseExistenceChecker,
) *UpdateSalesChannelUseCase {
	return &UpdateSalesChannelUseCase{db: db, repo: repo, connRepo: connRepo, warehouseRepo: warehouseRepo}
}

func (uc *UpdateSalesChannelUseCase) Execute(ctx context.Context, in UpdateSalesChannelInput) (*channelsDomain.SalesChannel, error) {
	var channel channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if err := validateSalesChannelInput(existing.Type, in.ConnectionID, in.EnrollmentMode, in.EnrollmentCategoryID, in.EnrollmentTag, in.PriceAdjustmentBasisPoints, in.QuantityBreaks); err != nil {
			return err
		}
		storefrontMode := existing.StorefrontMode
		if in.StorefrontMode != "" {
			if storefrontMode, err = channelsDomain.ResolveStorefrontMode(existing.Type, in.StorefrontMode); err != nil {
				return err
			}
		}
		storefrontLocale := existing.StorefrontLocale
		if in.StorefrontLocale != "" {
			if storefrontLocale, err = channelsDomain.ResolveStorefrontLocale(in.StorefrontLocale); err != nil {
				return err
			}
		}
		others := existing.OfferedStorefrontLocales()
		if in.StorefrontLocales != nil {
			others = *in.StorefrontLocales
		}
		storefrontLocales, err := channelsDomain.ResolveStorefrontLocales(storefrontLocale, others)
		if err != nil {
			return err
		}
		signInURL := existing.SignInURL
		if in.SignInURL != nil {
			if signInURL, err = channelsDomain.ResolveSignInURL(existing.Type, *in.SignInURL); err != nil {
				return err
			}
		}
		if in.ConnectionID != nil {
			conn, err := uc.connRepo.FindByID(ctx, *in.ConnectionID)
			if err != nil {
				return err
			}
			if string(conn.Provider) != string(existing.Type) {
				return channelsDomain.ErrChannelTypeConnectionMismatch
			}
		}
		if in.DefaultWarehouseID != nil {
			if _, err := uc.warehouseRepo.FindByID(ctx, *in.DefaultWarehouseID); err != nil {
				return err
			}
		}
		existing.Name = in.Name
		existing.ConnectionID = in.ConnectionID
		existing.EnrollmentMode = in.EnrollmentMode
		existing.EnrollmentCategoryID = in.EnrollmentCategoryID
		existing.EnrollmentTag = in.EnrollmentTag
		existing.PriceAdjustmentBasisPoints = in.PriceAdjustmentBasisPoints
		existing.QuantityBreaks = in.QuantityBreaks
		existing.DefaultWarehouseID = in.DefaultWarehouseID
		existing.StorefrontMode = storefrontMode
		existing.SignInURL = signInURL
		existing.StorefrontLocale = storefrontLocale
		existing.StorefrontLocales = storefrontLocales
		existing.Active = in.Active
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		channel = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateSalesChannel: %w", err)
	}
	return &channel, nil
}
