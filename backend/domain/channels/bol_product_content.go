package channels

import (
	"context"
	"time"
)

// BolProductContent holds per-(product, Bol connection) product-content
// fields Bol's Product Content API needs to move a listing from
// "Onvoldoende" (insufficient) to "Compleet" -- field list extracted
// directly from Bol's own product-content page for a real product in this
// catalog (category Babyhuidverzorging). Deliberately excludes
// Productgroep (Bol's category/GPC taxonomy, tracked as
// BolProductOverride.GPCCode) -- that needs mapping against Bol's real
// category tree, a separate piece of work. Every field is optional here
// (nil means "not filled in yet") even though Bol itself requires most of
// the "Basisinformatie" fields for a sellable listing -- this entity just
// stores whatever the merchant has entered so far; PushProductToBolUseCase
// decides what's actually required to submit.
//
// Connection-scoped, not provider-scoped, same reasoning as
// BolProductOverride -- a tenant could have more than one Bol connection.
type BolProductContent struct {
	ID           string
	TenantID     string
	ProductID    string
	ConnectionID string

	// Basisinformatie
	ManufacturerContact   *string // email or contact form URL
	ManufacturerAddress   *string
	ManufacturerName      *string
	ShelfLifeClosed       *string
	ShelfLifeAfterOpening *string
	Ingredients           *string
	ContentAmount         *string
	ContentUnit           *string
	CountryOfOrigin       *string
	Substance             *string
	InstructionLanguage   *string
	PackagingLanguage     *string

	// Optioneel
	ItemsPerPackage           *int32
	PackageInsertURL          *string
	IsGift                    *bool
	DermatologicallyTested    *bool
	HazardStatements          *string
	SkinType                  *string
	Hypoallergenic            *bool
	ContainsZincOxide         *bool
	ContentPerDispenserAmount *string
	ContentPerDispenserUnit   *string
	Brand                     *string
	FragranceFree             *bool
	PHNeutral                 *bool
	CoolingEffect             *bool
	PackageWidthMM            *int32
	PackageHeightMM           *int32
	PackageLengthMM           *int32
	PackageWeightG            *int32
	PrecautionStatements      *string
	Precautions               *string
	Warnings                  *string
	VideoURL                  *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type BolProductContentRepository interface {
	// FindByProductAndConnection returns nil, nil (not an error) when no
	// content has been filled in yet -- the common/default state, same
	// convention as BolProductOverrideRepository.
	FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*BolProductContent, error)
	Upsert(ctx context.Context, c *BolProductContent) error
}
