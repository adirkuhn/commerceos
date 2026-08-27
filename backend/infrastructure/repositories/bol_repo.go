package repositories

import (
	"context"
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
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
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
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "product_id"}, {Name: "connection_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"ean_override", "gpc_code", "condition_category", "fulfilment_method",
			"commission_override_basis_points", "reference_override", "custom_price_minor_units", "name_override", "updated_at",
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
		CreatedAt:                     m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
