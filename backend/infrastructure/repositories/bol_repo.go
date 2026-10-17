package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type bolProductOverrideModel struct {
	ID                            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                      string `gorm:"not null;type:uuid"`
	ProductID                     string `gorm:"not null;type:uuid"`
	ConnectionID                  string `gorm:"not null;type:uuid"`
	EANOverride                   *string
	GPCCode                       *string
	ConditionCategory             string `gorm:"not null"`
	FulfilmentMethod              string `gorm:"not null"`
	CommissionOverrideBasisPoints *int32
	ReferenceOverride             *string
	CustomPriceMinorUnits         *int64
	NameOverride                  *string
	BundlePricingEnabled          bool           `gorm:"not null"`
	BundleTiers                   bolBundleTiers `gorm:"column:bundle_tiers;type:jsonb;not null;default:'[]'"`
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
}

// bolBundleTiers is the JSONB bundle_tiers column, same whole-document
// convention as salesChannelQuantityBreaks.
type bolBundleTiers []channelsDomain.BolBundleTier

func (b bolBundleTiers) Value() (driver.Value, error) {
	if len(b) == 0 {
		return "[]", nil
	}
	out, err := json.Marshal([]channelsDomain.BolBundleTier(b))
	return string(out), err
}

func (b *bolBundleTiers) Scan(value any) error {
	var raw []byte
	switch v := value.(type) {
	case nil:
		*b = nil
		return nil
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("bolBundleTiers.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(raw, (*[]channelsDomain.BolBundleTier)(b))
}

func (bolProductOverrideModel) TableName() string { return "bol_product_overrides" }

type BolProductOverrideRepo struct{ db *gorm.DB }

func NewBolProductOverrideRepo(db *gorm.DB) *BolProductOverrideRepo {
	return &BolProductOverrideRepo{db: db}
}

func (r *BolProductOverrideRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("bol: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BolProductOverrideRepo) FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*channelsDomain.BolProductOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m bolProductOverrideModel
	err = tx.Where("product_id = ? AND connection_id = ?", productID, connectionID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bolProductOverrideRepo.FindByProductAndConnection: %w", err)
	}
	return toBolProductOverrideDomain(m), nil
}

// Upsert inserts or updates by the (product_id, connection_id) unique
// constraint (migration 095).
func (r *BolProductOverrideRepo) Upsert(ctx context.Context, o *channelsDomain.BolProductOverride) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := bolProductOverrideModel{
		TenantID: o.TenantID, ProductID: o.ProductID, ConnectionID: o.ConnectionID,
		EANOverride: o.EANOverride, GPCCode: o.GPCCode,
		ConditionCategory: string(o.ConditionCategory), FulfilmentMethod: string(o.FulfilmentMethod),
		CommissionOverrideBasisPoints: o.CommissionOverrideBasisPoints, ReferenceOverride: o.ReferenceOverride,
		CustomPriceMinorUnits: o.CustomPriceMinorUnits, NameOverride: o.NameOverride,
		BundlePricingEnabled: o.BundlePricingEnabled, BundleTiers: bolBundleTiers(o.BundleTiers),
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "product_id"}, {Name: "connection_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"ean_override", "gpc_code", "condition_category", "fulfilment_method",
			"commission_override_basis_points", "reference_override", "custom_price_minor_units", "name_override",
			"bundle_pricing_enabled", "bundle_tiers", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("bolProductOverrideRepo.Upsert: %w", err)
	}
	o.ID = m.ID
	return nil
}

func toBolProductOverrideDomain(m bolProductOverrideModel) *channelsDomain.BolProductOverride {
	return &channelsDomain.BolProductOverride{
		ID: m.ID, TenantID: m.TenantID, ProductID: m.ProductID, ConnectionID: m.ConnectionID,
		EANOverride: m.EANOverride, GPCCode: m.GPCCode,
		ConditionCategory:             channelsDomain.ConditionCategory(m.ConditionCategory),
		FulfilmentMethod:              channelsDomain.FulfilmentMethod(m.FulfilmentMethod),
		CommissionOverrideBasisPoints: m.CommissionOverrideBasisPoints,
		ReferenceOverride:             m.ReferenceOverride,
		CustomPriceMinorUnits:         m.CustomPriceMinorUnits,
		NameOverride:                  m.NameOverride,
		BundlePricingEnabled:          m.BundlePricingEnabled,
		BundleTiers:                   []channelsDomain.BolBundleTier(m.BundleTiers),
		CreatedAt:                     m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
