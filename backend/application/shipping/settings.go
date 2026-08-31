package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetShippingSettingsUseCase reads one connection's global defaults +
// every per-carrier override -- backs the settings page's "global and per
// carrier" configuration screen in one call.
type GetShippingSettingsUseCase struct {
	db           *gorm.DB
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository
	overrideRepo shippingDomain.ShippingCarrierOverrideRepository
}

func NewGetShippingSettingsUseCase(db *gorm.DB, settingsRepo shippingDomain.ShippingConnectionSettingsRepository, overrideRepo shippingDomain.ShippingCarrierOverrideRepository) *GetShippingSettingsUseCase {
	return &GetShippingSettingsUseCase{db: db, settingsRepo: settingsRepo, overrideRepo: overrideRepo}
}

type ShippingSettingsResult struct {
	Settings         *shippingDomain.ShippingConnectionSettings
	CarrierOverrides []shippingDomain.ShippingCarrierOverride
}

func (uc *GetShippingSettingsUseCase) Execute(ctx context.Context, connectionID string) (*ShippingSettingsResult, error) {
	var result ShippingSettingsResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		s, err := uc.settingsRepo.FindByConnectionID(ctx, connectionID)
		if err != nil {
			return err
		}
		result.Settings = s
		overrides, err := uc.overrideRepo.ListByConnection(ctx, connectionID)
		if err != nil {
			return err
		}
		result.CarrierOverrides = overrides
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getShippingSettings: %w", err)
	}
	return &result, nil
}

// UpdateShippingSettingsInput edits a connection's global defaults --
// ConnectionID must already have a settings row (created at connect time,
// see ConnectMyParcelUseCase).
type UpdateShippingSettingsInput struct {
	ConnectionID       string
	DefaultCarrier     shippingDomain.Carrier
	DefaultPackageType shippingDomain.PackageType
	DefaultLabelFormat string
	// DefaultLabelSize (shippingDomain.LabelSizeA4/LabelSizeA6) is
	// GetShipmentLabelUseCase's fallback physical page format -- see that
	// type's doc comment on why this isn't a per-shipment field.
	DefaultLabelSize         string
	DefaultSignatureRequired bool
	DefaultOnlyRecipient     bool
	InsureByDefault          bool
	MyParcelAccountID        *int
	MyParcelShopID           *int
	AutoPrintEnabled         bool
	// ExportOrderToMyParcel -- see domain/shipping.ShippingConnectionSettings'
	// doc comment on the field of the same name.
	ExportOrderToMyParcel bool
}

type UpdateShippingSettingsUseCase struct {
	db           *gorm.DB
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository
}

func NewUpdateShippingSettingsUseCase(db *gorm.DB, settingsRepo shippingDomain.ShippingConnectionSettingsRepository) *UpdateShippingSettingsUseCase {
	return &UpdateShippingSettingsUseCase{db: db, settingsRepo: settingsRepo}
}

func (uc *UpdateShippingSettingsUseCase) Execute(ctx context.Context, in UpdateShippingSettingsInput) (*shippingDomain.ShippingConnectionSettings, error) {
	settings := shippingDomain.ShippingConnectionSettings{
		ConnectionID: in.ConnectionID, DefaultCarrier: in.DefaultCarrier, DefaultPackageType: in.DefaultPackageType,
		DefaultLabelFormat: in.DefaultLabelFormat, DefaultLabelSize: in.DefaultLabelSize,
		DefaultSignatureRequired: in.DefaultSignatureRequired, DefaultOnlyRecipient: in.DefaultOnlyRecipient,
		InsureByDefault:   in.InsureByDefault,
		MyParcelAccountID: in.MyParcelAccountID, MyParcelShopID: in.MyParcelShopID, AutoPrintEnabled: in.AutoPrintEnabled,
		ExportOrderToMyParcel: in.ExportOrderToMyParcel,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		settings.CreatedAt, settings.UpdatedAt = now, now
		return uc.settingsRepo.Upsert(ctx, &settings)
	})
	if err != nil {
		return nil, fmt.Errorf("updateShippingSettings: %w", err)
	}
	return &settings, nil
}

// UpsertCarrierOverrideInput sets/clears one carrier's overrides on top of
// a connection's global defaults -- nil fields mean "no override, fall
// through to the connection setting" (see domain/shipping.ResolveOptions).
type UpsertCarrierOverrideInput struct {
	ConnectionID      string
	Carrier           shippingDomain.Carrier
	PackageType       *shippingDomain.PackageType
	SignatureRequired *bool
	OnlyRecipient     *bool
	InsureByDefault   *bool
}

type UpsertCarrierOverrideUseCase struct {
	db   *gorm.DB
	repo shippingDomain.ShippingCarrierOverrideRepository
}

func NewUpsertCarrierOverrideUseCase(db *gorm.DB, repo shippingDomain.ShippingCarrierOverrideRepository) *UpsertCarrierOverrideUseCase {
	return &UpsertCarrierOverrideUseCase{db: db, repo: repo}
}

func (uc *UpsertCarrierOverrideUseCase) Execute(ctx context.Context, in UpsertCarrierOverrideInput) (*shippingDomain.ShippingCarrierOverride, error) {
	override := shippingDomain.ShippingCarrierOverride{
		ConnectionID: in.ConnectionID, Carrier: in.Carrier,
		PackageType: in.PackageType, SignatureRequired: in.SignatureRequired, OnlyRecipient: in.OnlyRecipient,
		InsureByDefault: in.InsureByDefault,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		override.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		override.CreatedAt, override.UpdatedAt = now, now
		return uc.repo.Upsert(ctx, &override)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertCarrierOverride: %w", err)
	}
	return &override, nil
}
