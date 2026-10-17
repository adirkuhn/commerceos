package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// BolPricingDraft is unsaved Bol price settings from the product's Bol tab;
// when given it replaces the saved ones for the preview.
type BolPricingDraft struct {
	CustomPriceMinorUnits         *int64
	BundlePricingEnabled          bool
	BundleTiers                   []channelsDomain.BolBundleTier
	CommissionOverrideBasisPoints *int32
}

type PreviewBolPricingInput struct {
	ProductID    string
	ConnectionID string
	SKUID        string
	Draft        *BolPricingDraft
}

// BolPricingLevel is one price level as it would go to Bol.
type BolPricingLevel struct {
	channelsDomain.BolResolvedPrice
	// ProfitMinorUnits is what one unit leaves minus its cost price; nil
	// when the SKU has no cost price yet.
	ProfitMinorUnits *int64
}

type BolPricingPreview struct {
	EAN                    string
	RegularPriceMinorUnits int64
	CostPriceMinorUnits    *int64
	ProductVATBasisPoints  int64
	ShippingMinorUnits     int64
	Levels                 []BolPricingLevel
}

// PreviewBolPricingUseCase shows the Bol tab exactly the prices a push
// would send (see resolveBolPrices), for the saved settings or a draft.
type PreviewBolPricingUseCase struct {
	db           *gorm.DB
	connRepo     integrationsDomain.ConnectionRepository
	bolFactory   integrationsDomain.BolClientFactory
	encSecret    string
	skuRepo      skuFinder
	overrideRepo channelsDomain.BolProductOverrideRepository
	vatRates     vatRateResolver
}

func NewPreviewBolPricingUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	bolFactory integrationsDomain.BolClientFactory,
	encryptionSecret string,
	skuRepo skuFinder,
	overrideRepo channelsDomain.BolProductOverrideRepository,
	vatRates vatRateResolver,
) *PreviewBolPricingUseCase {
	return &PreviewBolPricingUseCase{
		db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret,
		skuRepo: skuRepo, overrideRepo: overrideRepo, vatRates: vatRates,
	}
}

func (uc *PreviewBolPricingUseCase) Execute(ctx context.Context, in PreviewBolPricingInput) (*BolPricingPreview, error) {
	if in.Draft != nil && in.Draft.BundlePricingEnabled {
		if len(in.Draft.BundleTiers) == 0 {
			in.Draft.BundleTiers = channelsDomain.DefaultBolBundleTiers()
		}
		if err := channelsDomain.ValidateBolBundleTiers(in.Draft.BundleTiers); err != nil {
			return nil, fmt.Errorf("previewBolPricing: %w", err)
		}
	}

	var conn integrationsDomain.Connection
	var sku pmsDomain.SKU
	var override *channelsDomain.BolProductOverride
	var vat int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		s, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}
		sku = *s
		if override, err = uc.overrideRepo.FindByProductAndConnection(ctx, in.ProductID, in.ConnectionID); err != nil {
			return err
		}
		vat, err = uc.vatRates.RateBasisPoints(ctx, sku.TaxClassID, bolVATCountry)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("previewBolPricing: %w", err)
	}

	ean, err := bolEAN(sku, override)
	if err != nil {
		return nil, fmt.Errorf("previewBolPricing: %w", err)
	}
	cfg := bolPricingConfigOf(override)
	if in.Draft != nil {
		cfg = bolPricingConfig(*in.Draft)
	}

	secret, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("previewBolPricing: decrypt secret: %w", err)
	}
	client, err := uc.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("previewBolPricing: build client: %w", err)
	}

	priceIn := channelsDomain.BolPriceInputs{
		RegularPriceMinorUnits: sku.PriceMinorUnits, ProductVATBasisPoints: vat, ShippingMinorUnits: conn.BolShippingCostMinorUnits,
	}
	resolved, err := resolveBolPrices(ctx, client, ean, priceIn, cfg)
	if err != nil {
		return nil, fmt.Errorf("previewBolPricing: %w", err)
	}

	levels := make([]BolPricingLevel, len(resolved))
	for i, r := range resolved {
		levels[i] = BolPricingLevel{BolResolvedPrice: r}
		if sku.CostPriceMinorUnits != nil {
			profit := r.Economics.NetMinorUnits - *sku.CostPriceMinorUnits
			levels[i].ProfitMinorUnits = &profit
		}
	}
	return &BolPricingPreview{
		EAN: ean, RegularPriceMinorUnits: sku.PriceMinorUnits, CostPriceMinorUnits: sku.CostPriceMinorUnits,
		ProductVATBasisPoints: vat, ShippingMinorUnits: conn.BolShippingCostMinorUnits, Levels: levels,
	}, nil
}
