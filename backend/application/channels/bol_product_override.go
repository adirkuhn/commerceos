package channels

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetBolProductOverrideUseCase struct {
	db   *gorm.DB
	repo channelsDomain.BolProductOverrideRepository
}

func NewGetBolProductOverrideUseCase(db *gorm.DB, repo channelsDomain.BolProductOverrideRepository) *GetBolProductOverrideUseCase {
	return &GetBolProductOverrideUseCase{db: db, repo: repo}
}

// Execute returns nil, nil when no override exists yet -- the common case
// for a product with no Bol-specific configuration.
func (uc *GetBolProductOverrideUseCase) Execute(ctx context.Context, productID, connectionID string) (*channelsDomain.BolProductOverride, error) {
	var result *channelsDomain.BolProductOverride
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.repo.FindByProductAndConnection(ctx, productID, connectionID)
		if err != nil {
			return err
		}
		result = o
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getBolProductOverride: %w", err)
	}
	return result, nil
}

type UpsertBolProductOverrideInput struct {
	ProductID                     string
	ConnectionID                  string
	EANOverride                   *string
	GPCCode                       *string
	ConditionCategory             string
	FulfilmentMethod              string
	CommissionOverrideBasisPoints *int32
	ReferenceOverride             *string
}

type UpsertBolProductOverrideUseCase struct {
	db   *gorm.DB
	repo channelsDomain.BolProductOverrideRepository
}

func NewUpsertBolProductOverrideUseCase(db *gorm.DB, repo channelsDomain.BolProductOverrideRepository) *UpsertBolProductOverrideUseCase {
	return &UpsertBolProductOverrideUseCase{db: db, repo: repo}
}

func (uc *UpsertBolProductOverrideUseCase) Execute(ctx context.Context, in UpsertBolProductOverrideInput) (*channelsDomain.BolProductOverride, error) {
	condition := channelsDomain.ConditionCategory(in.ConditionCategory)
	if condition == "" {
		condition = channelsDomain.BolConditionNew
	}
	fulfilment := channelsDomain.FulfilmentMethod(in.FulfilmentMethod)
	if fulfilment == "" {
		fulfilment = channelsDomain.BolFulfilmentFBR
	}

	override := channelsDomain.BolProductOverride{
		ProductID: in.ProductID, ConnectionID: in.ConnectionID,
		EANOverride: in.EANOverride, GPCCode: in.GPCCode,
		ConditionCategory: condition, FulfilmentMethod: fulfilment,
		CommissionOverrideBasisPoints: in.CommissionOverrideBasisPoints,
		ReferenceOverride:             in.ReferenceOverride,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		override.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		if override.CreatedAt.IsZero() {
			override.CreatedAt = now
		}
		override.UpdatedAt = now
		return uc.repo.Upsert(ctx, &override)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertBolProductOverride: %w", err)
	}
	return &override, nil
}
