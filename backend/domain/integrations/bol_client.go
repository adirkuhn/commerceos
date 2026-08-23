package integrations

import "context"

// BolOfferInput is what PushProductToBolUseCase needs to create/update one
// Bol offer -- domain-owned shape, kept separate from infrastructure/bol's
// own HTTP request struct so domain/application never import infrastructure
// types (see CLAUDE.md's layering rule).
type BolOfferInput struct {
	EAN                    string
	UnknownProductTitle    string
	EconomicOperatorID     string
	Reference              string
	ConditionCategory      string
	FulfilmentMethod       string
	UnitPrice              float64
	StockAmount            int
	StockManagedByRetailer bool
}

// BolProductContentInput is what's needed to submit/update Bol catalog
// content for an EAN that isn't yet known to Bol -- only used when
// CreateOffer reports an unknown EAN.
type BolProductContentInput struct {
	EAN         string
	GPCCode     string
	Title       string
	Description string
	ImageURLs   []string
}

type BolCommission struct {
	EAN         string
	FixedAmount float64
	Percentage  float64
}

// BolClient is one Bol connection's credentials + endpoints, adapted to
// exactly what this codebase's use cases need (ISP) -- mirrors WooClient's
// role. Implemented by infrastructure/bol.
type BolClient interface {
	CreateOffer(ctx context.Context, in BolOfferInput) (offerID string, err error)
	UpdateOffer(ctx context.Context, offerID string, in BolOfferInput) error
	// IsUnknownEAN classifies an error CreateOffer/UpdateOffer returned --
	// true means "this EAN isn't in Bol's catalog yet, submit product
	// content first," false means some other failure.
	IsUnknownEAN(err error) bool
	SubmitProductContent(ctx context.Context, in BolProductContentInput) error
	GetCommission(ctx context.Context, ean string, unitPrice float64) (BolCommission, error)
}

// BolClientFactory builds a BolClient scoped to one connection's credentials.
type BolClientFactory interface {
	NewClient(clientID, clientSecret string) (BolClient, error)
}
