package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type shippingConnectionSettingsModel struct {
	ConnectionID             string `gorm:"primaryKey;type:uuid"`
	TenantID                 string `gorm:"not null;type:uuid"`
	DefaultCarrier           int
	DefaultPackageType       int
	DefaultLabelFormat       string
	DefaultLabelSize         string
	DefaultSignatureRequired bool
	DefaultOnlyRecipient     bool
	InsureByDefault          bool
	// MyParcelAccountID/MyParcelShopID need explicit column tags -- GORM's
	// default namer splits "MyParcel" into two words ("my_parcel_..."),
	// but migration 096/111 named these columns "myparcel_..." (matching
	// this codebase's other MyParcel-prefixed names, e.g.
	// ShipmentRepo's ExternalShipmentID `gorm:"column:external_shipment_id"`)
	// -- without the tag, Upsert fails with "column my_parcel_account_id
	// does not exist" (caught 2026-08-27).
	MyParcelAccountID     *int `gorm:"column:myparcel_account_id"`
	MyParcelShopID        *int `gorm:"column:myparcel_shop_id"`
	AutoPrintEnabled      bool
	ExportOrderToMyParcel bool `gorm:"column:export_order_to_myparcel"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (shippingConnectionSettingsModel) TableName() string { return "shipping_connection_settings" }

type ShippingConnectionSettingsRepo struct{ db *gorm.DB }

func NewShippingConnectionSettingsRepo(db *gorm.DB) *ShippingConnectionSettingsRepo {
	return &ShippingConnectionSettingsRepo{db: db}
}

func (r *ShippingConnectionSettingsRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shippingConnectionSettingsRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ShippingConnectionSettingsRepo) FindByConnectionID(ctx context.Context, connectionID string) (*shippingDomain.ShippingConnectionSettings, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shippingConnectionSettingsModel
	err = tx.Where("connection_id = ?", connectionID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("shippingConnectionSettingsRepo.FindByConnectionID: %w", err)
	}
	return &shippingDomain.ShippingConnectionSettings{
		ConnectionID: m.ConnectionID, TenantID: m.TenantID,
		DefaultCarrier: shippingDomain.Carrier(m.DefaultCarrier), DefaultPackageType: shippingDomain.PackageType(m.DefaultPackageType),
		DefaultLabelFormat: m.DefaultLabelFormat, DefaultLabelSize: m.DefaultLabelSize,
		DefaultSignatureRequired: m.DefaultSignatureRequired, DefaultOnlyRecipient: m.DefaultOnlyRecipient,
		InsureByDefault:   m.InsureByDefault,
		MyParcelAccountID: m.MyParcelAccountID, MyParcelShopID: m.MyParcelShopID, AutoPrintEnabled: m.AutoPrintEnabled,
		ExportOrderToMyParcel: m.ExportOrderToMyParcel,
		CreatedAt:             m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *ShippingConnectionSettingsRepo) Upsert(ctx context.Context, s *shippingDomain.ShippingConnectionSettings) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := shippingConnectionSettingsModel{
		ConnectionID: s.ConnectionID, TenantID: s.TenantID,
		DefaultCarrier: int(s.DefaultCarrier), DefaultPackageType: int(s.DefaultPackageType),
		DefaultLabelFormat: s.DefaultLabelFormat, DefaultLabelSize: s.DefaultLabelSize,
		DefaultSignatureRequired: s.DefaultSignatureRequired, DefaultOnlyRecipient: s.DefaultOnlyRecipient,
		InsureByDefault:   s.InsureByDefault,
		MyParcelAccountID: s.MyParcelAccountID, MyParcelShopID: s.MyParcelShopID, AutoPrintEnabled: s.AutoPrintEnabled,
		ExportOrderToMyParcel: s.ExportOrderToMyParcel,
		CreatedAt:             s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "connection_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"default_carrier", "default_package_type", "default_label_format", "default_label_size",
			"default_signature_required", "default_only_recipient",
			"insure_by_default", "myparcel_account_id", "myparcel_shop_id", "auto_print_enabled",
			"export_order_to_myparcel", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("shippingConnectionSettingsRepo.Upsert: %w", err)
	}
	return nil
}

// --- ShippingCarrierOverride ---

type shippingCarrierOverrideModel struct {
	ID                  string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID            string `gorm:"not null;type:uuid"`
	ConnectionID        string `gorm:"not null;type:uuid"`
	Carrier             int
	PackageType         *int
	SignatureRequired   *bool
	OnlyRecipient       *bool
	InsureByDefault     *bool
	TrackingURLTemplate *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (shippingCarrierOverrideModel) TableName() string { return "shipping_carrier_overrides" }

type ShippingCarrierOverrideRepo struct{ db *gorm.DB }

func NewShippingCarrierOverrideRepo(db *gorm.DB) *ShippingCarrierOverrideRepo {
	return &ShippingCarrierOverrideRepo{db: db}
}

func (r *ShippingCarrierOverrideRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shippingCarrierOverrideRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ShippingCarrierOverrideRepo) ListByConnection(ctx context.Context, connectionID string) ([]shippingDomain.ShippingCarrierOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var ms []shippingCarrierOverrideModel
	if err := tx.Where("connection_id = ?", connectionID).Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("shippingCarrierOverrideRepo.ListByConnection: %w", err)
	}
	out := make([]shippingDomain.ShippingCarrierOverride, len(ms))
	for i, m := range ms {
		out[i] = toCarrierOverrideDomain(m)
	}
	return out, nil
}

func (r *ShippingCarrierOverrideRepo) FindByConnectionAndCarrier(ctx context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shippingCarrierOverrideModel
	err = tx.Where("connection_id = ? AND carrier = ?", connectionID, int(carrier)).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("shippingCarrierOverrideRepo.FindByConnectionAndCarrier: %w", err)
	}
	o := toCarrierOverrideDomain(m)
	return &o, nil
}

func (r *ShippingCarrierOverrideRepo) Upsert(ctx context.Context, o *shippingDomain.ShippingCarrierOverride) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := shippingCarrierOverrideModel{
		TenantID: o.TenantID, ConnectionID: o.ConnectionID, Carrier: int(o.Carrier),
		PackageType: intPtrFromPackageType(o.PackageType), SignatureRequired: o.SignatureRequired, OnlyRecipient: o.OnlyRecipient,
		InsureByDefault: o.InsureByDefault, TrackingURLTemplate: o.TrackingURLTemplate,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "connection_id"}, {Name: "carrier"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"package_type", "signature_required", "only_recipient", "insure_by_default", "tracking_url_template", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("shippingCarrierOverrideRepo.Upsert: %w", err)
	}
	o.ID = m.ID
	return nil
}

func intPtrFromPackageType(pt *shippingDomain.PackageType) *int {
	if pt == nil {
		return nil
	}
	v := int(*pt)
	return &v
}

func toCarrierOverrideDomain(m shippingCarrierOverrideModel) shippingDomain.ShippingCarrierOverride {
	var pt *shippingDomain.PackageType
	if m.PackageType != nil {
		v := shippingDomain.PackageType(*m.PackageType)
		pt = &v
	}
	return shippingDomain.ShippingCarrierOverride{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID, Carrier: shippingDomain.Carrier(m.Carrier),
		PackageType: pt, SignatureRequired: m.SignatureRequired, OnlyRecipient: m.OnlyRecipient,
		InsureByDefault: m.InsureByDefault, TrackingURLTemplate: m.TrackingURLTemplate,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
