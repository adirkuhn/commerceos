package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestGetBolCommission_fetchesLiveCommissionForSKUEAN(t *testing.T) {
	sku := pmsDomain.SKU{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}
	skuRepo := &fakeSKUByIDRepo{skus: map[string]pmsDomain.SKU{"sku-1": sku}}
	conn := bolConn()
	connRepo := &stubConnectionRepo{conn: &conn}
	overrideRepo := &fakeBolOverrideRepo{}

	var gotEAN string
	var gotPrice float64
	client := &fakeBolClient{}
	client.getCommissionFn = func(_ context.Context, ean string, price float64) (integrationsDomain.BolCommission, error) {
		gotEAN, gotPrice = ean, price
		return integrationsDomain.BolCommission{EAN: ean, FixedAmount: 0.30, Percentage: 12.5}, nil
	}

	uc := appchannels.NewGetBolCommissionUseCase(newMockGormDB(t, 1), connRepo, &fakeBolClientFactory{client: client}, testEncryptionSecret, skuRepo, overrideRepo)
	result, err := uc.Execute(context.Background(), appchannels.GetBolCommissionInput{ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1"})

	require.NoError(t, err)
	assert.Equal(t, "0630509892495", gotEAN)
	assert.InDelta(t, 19.99, gotPrice, 0.001)
	assert.InDelta(t, 0.30, result.FixedAmount, 0.001)
	assert.InDelta(t, 12.5, result.Percentage, 0.001)
	assert.False(t, result.Overridden)
}

func TestGetBolCommission_appliesOverrideInsteadOfLiveFetch(t *testing.T) {
	sku := pmsDomain.SKU{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}
	skuRepo := &fakeSKUByIDRepo{skus: map[string]pmsDomain.SKU{"sku-1": sku}}
	conn := bolConn()
	connRepo := &stubConnectionRepo{conn: &conn}
	basisPoints := int32(1500) // 15.00%
	overrideRepo := &fakeBolOverrideRepo{override: &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1", CommissionOverrideBasisPoints: &basisPoints,
	}}

	client := &fakeBolClient{}
	client.getCommissionFn = func(_ context.Context, _ string, _ float64) (integrationsDomain.BolCommission, error) {
		t.Fatal("must not call the live commission API when an override is set")
		return integrationsDomain.BolCommission{}, nil
	}

	uc := appchannels.NewGetBolCommissionUseCase(newMockGormDB(t, 1), connRepo, &fakeBolClientFactory{client: client}, testEncryptionSecret, skuRepo, overrideRepo)
	result, err := uc.Execute(context.Background(), appchannels.GetBolCommissionInput{ProductID: "prod-1", ConnectionID: "conn-1", SKUID: "sku-1"})

	require.NoError(t, err)
	assert.InDelta(t, 15.0, result.Percentage, 0.001)
	assert.True(t, result.Overridden)
}
