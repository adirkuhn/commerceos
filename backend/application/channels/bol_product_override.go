package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/idgen"
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
	CustomPriceMinorUnits         *int64
	NameOverride                  *string
	// BundlePricingEnabled turns Bol bulk prices on; switching it on with
	// no BundleTiers starts with the 2/4/6/8 defaults. Tiers are kept while
	// it is off.
	BundlePricingEnabled bool
	BundleTiers          []channelsDomain.BolBundleTier
}

type UpsertBolProductOverrideUseCase struct {
	db         *gorm.DB
	repo       channelsDomain.BolProductOverrideRepository
	outboxRepo outbox.Repository
}

func NewUpsertBolProductOverrideUseCase(db *gorm.DB, repo channelsDomain.BolProductOverrideRepository, outboxRepo outbox.Repository) *UpsertBolProductOverrideUseCase {
	return &UpsertBolProductOverrideUseCase{db: db, repo: repo, outboxRepo: outboxRepo}
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

	tiers := in.BundleTiers
	if in.BundlePricingEnabled && len(tiers) == 0 {
		tiers = channelsDomain.DefaultBolBundleTiers()
	}
	if err := channelsDomain.ValidateBolBundleTiers(tiers); err != nil {
		return nil, fmt.Errorf("upsertBolProductOverride: %w", err)
	}

	override := channelsDomain.BolProductOverride{
		ProductID: in.ProductID, ConnectionID: in.ConnectionID,
		EANOverride: in.EANOverride, GPCCode: in.GPCCode,
		ConditionCategory: condition, FulfilmentMethod: fulfilment,
		CommissionOverrideBasisPoints: in.CommissionOverrideBasisPoints,
		ReferenceOverride:             in.ReferenceOverride,
		CustomPriceMinorUnits:         in.CustomPriceMinorUnits,
		NameOverride:                  in.NameOverride,
		BundlePricingEnabled:          in.BundlePricingEnabled,
		BundleTiers:                   tiers,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		override.TenantID = tenant.IDFromContext(ctx)
		now := time.Now()
		if override.CreatedAt.IsZero() {
			override.CreatedAt = now
		}
		override.UpdatedAt = now
		if err := uc.repo.Upsert(ctx, &override); err != nil {
			return err
		}

		// Raises the same pms.product.changed event a normal product edit
		// does, atomically with the write above (mandatory outbox pattern,
		// see CLAUDE.md) -- consumers re-push the product to every channel
		// it's already linked to. Without this, saving the Bol tab's Custom
		// price wrote CustomPriceMinorUnits to the DB but Bol's offer was
		// never told about it (caught 2026-08-27: price saved successfully,
		// never synced). Mirrors UpsertBolProductContentUseCase.
		event := events.ProductChanged{
			Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID: tenant.IDFromContext(ctx), ProductID: override.ProductID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("upsertBolProductOverride: %w", err)
	}
	return &override, nil
}
