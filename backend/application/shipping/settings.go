package shipping

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	outbox       eventrecord.Enqueuer
}

func NewUpdateShippingSettingsUseCase(db *gorm.DB, settingsRepo shippingDomain.ShippingConnectionSettingsRepository, outbox eventrecord.Enqueuer) *UpdateShippingSettingsUseCase {
	return &UpdateShippingSettingsUseCase{db: db, settingsRepo: settingsRepo, outbox: outbox}
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
		before, err := uc.settingsRepo.FindByConnectionID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		if before == nil {
			before = &shippingDomain.ShippingConnectionSettings{}
		}
		if err := uc.settingsRepo.Upsert(ctx, &settings); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ShippingSettingsUpdated, in.ConnectionID, shippingSettingsChanges(*before, settings))
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
	ConnectionID        string
	Carrier             shippingDomain.Carrier
	PackageType         *shippingDomain.PackageType
	SignatureRequired   *bool
	OnlyRecipient       *bool
	InsureByDefault     *bool
	TrackingURLTemplate *string
}

type UpsertCarrierOverrideUseCase struct {
	db     *gorm.DB
	repo   shippingDomain.ShippingCarrierOverrideRepository
	outbox eventrecord.Enqueuer
}

func NewUpsertCarrierOverrideUseCase(db *gorm.DB, repo shippingDomain.ShippingCarrierOverrideRepository, outbox eventrecord.Enqueuer) *UpsertCarrierOverrideUseCase {
	return &UpsertCarrierOverrideUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *UpsertCarrierOverrideUseCase) Execute(ctx context.Context, in UpsertCarrierOverrideInput) (*shippingDomain.ShippingCarrierOverride, error) {
	override := shippingDomain.ShippingCarrierOverride{
		ConnectionID: in.ConnectionID, Carrier: in.Carrier,
		PackageType: in.PackageType, SignatureRequired: in.SignatureRequired, OnlyRecipient: in.OnlyRecipient,
		InsureByDefault: in.InsureByDefault, TrackingURLTemplate: in.TrackingURLTemplate,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		override.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		override.CreatedAt, override.UpdatedAt = now, now
		before, err := uc.repo.FindByConnectionAndCarrier(ctx, in.ConnectionID, in.Carrier)
		if err != nil {
			return err
		}
		if before == nil {
			before = &shippingDomain.ShippingCarrierOverride{}
		}
		if err := uc.repo.Upsert(ctx, &override); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.CarrierOverrideUpdated, in.ConnectionID,
			eventrecord.Data{"carrier": int(in.Carrier)}, eventrecord.NewChanges().
				Set("package_type", before.PackageType, override.PackageType).
				Set("signature_required", before.SignatureRequired, override.SignatureRequired).
				Set("only_recipient", before.OnlyRecipient, override.OnlyRecipient).
				Set("insure_by_default", before.InsureByDefault, override.InsureByDefault).
				Set("tracking_url_template", before.TrackingURLTemplate, override.TrackingURLTemplate))
	})
	if err != nil {
		return nil, fmt.Errorf("upsertCarrierOverride: %w", err)
	}
	return &override, nil
}

func shippingSettingsChanges(before, after shippingDomain.ShippingConnectionSettings) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("default_carrier", int(before.DefaultCarrier), int(after.DefaultCarrier)).
		Set("default_package_type", int(before.DefaultPackageType), int(after.DefaultPackageType)).
		Set("default_label_format", before.DefaultLabelFormat, after.DefaultLabelFormat).
		Set("default_label_size", before.DefaultLabelSize, after.DefaultLabelSize).
		Set("default_signature_required", before.DefaultSignatureRequired, after.DefaultSignatureRequired).
		Set("default_only_recipient", before.DefaultOnlyRecipient, after.DefaultOnlyRecipient).
		Set("insure_by_default", before.InsureByDefault, after.InsureByDefault).
		Set("myparcel_account_id", before.MyParcelAccountID, after.MyParcelAccountID).
		Set("myparcel_shop_id", before.MyParcelShopID, after.MyParcelShopID).
		Set("auto_print_enabled", before.AutoPrintEnabled, after.AutoPrintEnabled).
		Set("export_order_to_myparcel", before.ExportOrderToMyParcel, after.ExportOrderToMyParcel)
}
