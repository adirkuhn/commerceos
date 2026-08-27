package channels

import (
	"context"
	"time"
)

const (
	BolConditionNew ConditionCategory = "NEW"

	BolFulfilmentFBR FulfilmentMethod = "FBR"
)

type ConditionCategory string
type FulfilmentMethod string

// BolProductOverride holds per-(product, Bol connection) fields that either
// aren't derivable from pms.Product/SKU at all (GPCCode, the Bol category
// taxonomy id -- pms.Category has nothing equivalent) or need to be
// overridable per channel (EANOverride, when pms.SKU.Barcode isn't the EAN
// Bol should list under; CommissionOverrideBasisPoints, when the merchant
// wants to book a different margin assumption than whatever Bol's live
// Commissions API returned). Connection-scoped, not provider-scoped, same
// reasoning as ExternalCategoryLink -- a tenant could have more than one Bol
// connection (e.g. two economic operator ids) with different mappings.
//
// A missing row is the common case (no product has Bol-specific
// configuration yet) and isn't an error -- see
// BolProductOverrideRepository.FindByProductAndConnection.
type BolProductOverride struct {
	ID           string
	TenantID     string
	ProductID    string
	ConnectionID string
	// EANOverride wins over pms.SKU.Barcode when set. nil means "use the
	// SKU's own barcode as the EAN."
	EANOverride *string
	// GPCCode is Bol's product-category classification id, required by the
	// Product Content API only when the EAN doesn't already exist in Bol's
	// catalog (see PushProductToBolUseCase). nil until the merchant fills
	// it in -- pushing a genuinely-new EAN without one fails with a clear
	// error rather than guessing a category.
	GPCCode *string
	// ConditionCategory/FulfilmentMethod default to NEW/FBR (self-ship) --
	// v1 deliberately doesn't support FBB (Fulfilment by Bol), a materially
	// different flow (Bol's own warehouse).
	ConditionCategory ConditionCategory
	FulfilmentMethod  FulfilmentMethod
	// CommissionOverrideBasisPoints overrides the live-fetched Bol
	// commission in the profit calculator (see application/channels.
	// GetBolCommissionUseCase) -- nil means "use whatever Bol's API
	// returned for this EAN/price," since commission is price-bracket
	// dependent and can't be meaningfully defaulted otherwise.
	CommissionOverrideBasisPoints *int32
	// ReferenceOverride is Bol's offer.reference field (merchant's own
	// free-text identifier, max 100 chars per Bol's docs) -- optional,
	// nil means omit it from the offer payload.
	ReferenceOverride *string
	// CustomPriceMinorUnits is the merchant's own sell price for this Bol
	// offer, minor currency units. nil means "use pms.SKU.PriceMinorUnits" --
	// same fallback convention as EANOverride. Feeds both the offer pushed
	// to Bol (PushProductToBolUseCase) and the price-bracket-dependent
	// commission lookup (GetBolCommissionUseCase).
	CustomPriceMinorUnits *int64
	// NameOverride wins over pms.Product.Name when set, as the Title
	// submitted to Bol's Product Content API (PushProductToBolUseCase) --
	// same fallback convention as EANOverride/CustomPriceMinorUnits.
	// Bol-only: no other channel reads this field, so a merchant can list a
	// different name on Bol without touching the product's real name (which
	// still feeds Woo and everywhere else in CommerceOS).
	NameOverride *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BolProductOverrideRepository interface {
	// FindByProductAndConnection returns nil, nil (not an error) when no
	// override exists yet for this (product, connection) pair -- the
	// common/default state, same convention as
	// AIConnectionRepository.Get.
	FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*BolProductOverride, error)
	Upsert(ctx context.Context, o *BolProductOverride) error
}
