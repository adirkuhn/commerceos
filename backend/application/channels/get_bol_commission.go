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

// skuFinder is the one method GetBolCommissionUseCase needs from
// pmsDomain.SKURepository (ISP).
type skuFinder interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

type GetBolCommissionInput struct {
	ProductID    string
	ConnectionID string
	SKUID        string
}

type BolCommissionResult struct {
	EAN         string
	FixedAmount float64
	Percentage  float64
	// Overridden is true when the result came from
	// BolProductOverride.CommissionOverrideBasisPoints instead of a live
	// Bol API call -- the profit calculator surfaces this so the merchant
	// knows they're looking at their own assumption, not what Bol
	// currently charges.
	Overridden bool
}

// GetBolCommissionUseCase backs the profit calculator's commission input --
// live per-EAN/price from Bol's Commissions API by default (price-bracket
// dependent, so it's fetched fresh rather than cached), or the merchant's
// own BolProductOverride.CommissionOverrideBasisPoints when one is set.
type GetBolCommissionUseCase struct {
	db           *gorm.DB
	connRepo     integrationsDomain.ConnectionRepository
	bolFactory   integrationsDomain.BolClientFactory
	encSecret    string
	skuRepo      skuFinder
	overrideRepo channelsDomain.BolProductOverrideRepository
}

func NewGetBolCommissionUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	bolFactory integrationsDomain.BolClientFactory,
	encryptionSecret string,
	skuRepo skuFinder,
	overrideRepo channelsDomain.BolProductOverrideRepository,
) *GetBolCommissionUseCase {
	return &GetBolCommissionUseCase{
		db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret,
		skuRepo: skuRepo, overrideRepo: overrideRepo,
	}
}

func (uc *GetBolCommissionUseCase) Execute(ctx context.Context, in GetBolCommissionInput) (*BolCommissionResult, error) {
	var conn integrationsDomain.Connection
	var sku pmsDomain.SKU
	var override *channelsDomain.BolProductOverride

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

		override, err = uc.overrideRepo.FindByProductAndConnection(ctx, in.ProductID, in.ConnectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getBolCommission: %w", err)
	}

	ean := sku.Barcode
	if override != nil && override.EANOverride != nil && *override.EANOverride != "" {
		ean = *override.EANOverride
	}
	if ean == "" {
		return nil, fmt.Errorf("getBolCommission: %w", channelsDomain.ErrBolMissingEAN)
	}

	if override != nil && override.CommissionOverrideBasisPoints != nil {
		return &BolCommissionResult{
			EAN: ean, Percentage: float64(*override.CommissionOverrideBasisPoints) / 100, Overridden: true,
		}, nil
	}

	secret, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("getBolCommission: decrypt secret: %w", err)
	}
	client, err := uc.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("getBolCommission: build client: %w", err)
	}

	unitPrice := float64(sku.PriceMinorUnits) / 100
	commission, err := client.GetCommission(ctx, ean, unitPrice)
	if err != nil {
		return nil, fmt.Errorf("getBolCommission: %w", err)
	}
	return &BolCommissionResult{EAN: commission.EAN, FixedAmount: commission.FixedAmount, Percentage: commission.Percentage}, nil
}
