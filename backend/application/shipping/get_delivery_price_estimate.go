package shipping

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// GetDeliveryPriceEstimateInput drives GetDeliveryPriceEstimateUseCase --
// AddressConfirmModal's "how much will this label cost" preview, for the
// one slice of the world MyParcel actually prices before a shipment exists.
type GetDeliveryPriceEstimateInput struct {
	ConnectionID string
	CC           string
	PostalCode   string
	Number       string
	Carrier      shippingDomain.Carrier
}

// GetDeliveryPriceEstimateUseCase is AddressConfirmModal's pre-confirm cost
// preview: resolves the connection's MyParcel API key and asks MyParcel's
// GET /delivery_options for this address/carrier's priced delivery slots.
// Only ever meaningful for NL/BE -- see
// domain/shipping.ErrDeliveryEstimateUnsupportedCountry's doc comment for
// why every other country is rejected here rather than sent upstream.
// Read-only: no persistence, no outbox, this never changes any state.
type GetDeliveryPriceEstimateUseCase struct {
	db              *gorm.DB
	connRepo        integrationsDomain.ConnectionRepository
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
}

func NewGetDeliveryPriceEstimateUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, myParcelFactory integrationsDomain.MyParcelClientFactory, encryptionSecret string) *GetDeliveryPriceEstimateUseCase {
	return &GetDeliveryPriceEstimateUseCase{db: db, connRepo: connRepo, myParcelFactory: myParcelFactory, encSecret: encryptionSecret}
}

func (uc *GetDeliveryPriceEstimateUseCase) Execute(ctx context.Context, in GetDeliveryPriceEstimateInput) ([]shippingDomain.DeliveryPriceOption, error) {
	cc := strings.ToUpper(strings.TrimSpace(in.CC))
	if cc != "NL" && cc != "BE" {
		return nil, fmt.Errorf("getDeliveryPriceEstimate: %w", shippingDomain.ErrDeliveryEstimateUnsupportedCountry)
	}

	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getDeliveryPriceEstimate: %w", err)
	}

	apiKey, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("getDeliveryPriceEstimate: decrypt api key: %w", err)
	}
	client, err := uc.myParcelFactory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("getDeliveryPriceEstimate: build client: %w", err)
	}

	options, err := client.GetDeliveryPriceEstimate(ctx, integrationsDomain.DeliveryPriceEstimateInput{
		CC: cc, PostalCode: in.PostalCode, Number: in.Number, Carrier: int(in.Carrier),
	})
	if err != nil {
		return nil, fmt.Errorf("getDeliveryPriceEstimate: %w", err)
	}

	out := make([]shippingDomain.DeliveryPriceOption, len(options))
	for i, o := range options {
		out[i] = shippingDomain.DeliveryPriceOption{PriceComment: o.PriceComment, AmountMinorUnits: o.AmountMinorUnits, Currency: o.Currency}
	}
	return out, nil
}
