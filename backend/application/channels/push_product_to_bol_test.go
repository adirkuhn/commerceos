package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// --- Bol-specific fakes (BolClient/BolClientFactory/BolProductOverrideRepository have no other test double in this package yet) ---

type fakeBolClient struct {
	createOfferFn    func(ctx context.Context, in integrationsDomain.BolOfferInput) (string, error)
	updateOfferCalls []integrationsDomain.BolOfferInput
	contentCalls     []integrationsDomain.BolProductContentInput
	unknownEANErr    error
	getCommissionFn  func(ctx context.Context, ean string, unitPrice float64) (integrationsDomain.BolCommission, error)
	listOrdersFn     func(ctx context.Context, page int, status string) ([]integrationsDomain.BolOrder, error)
	getOrderFn       func(ctx context.Context, orderID string) (*integrationsDomain.BolOrder, error)
}

func (f *fakeBolClient) CreateOffer(_ context.Context, in integrationsDomain.BolOfferInput) (string, error) {
	return f.createOfferFn(context.Background(), in)
}
func (f *fakeBolClient) UpdateOffer(_ context.Context, _ string, in integrationsDomain.BolOfferInput) error {
	f.updateOfferCalls = append(f.updateOfferCalls, in)
	return nil
}
func (f *fakeBolClient) IsUnknownEAN(err error) bool {
	return f.unknownEANErr != nil && errors.Is(err, f.unknownEANErr)
}
func (f *fakeBolClient) SubmitProductContent(_ context.Context, in integrationsDomain.BolProductContentInput) error {
	f.contentCalls = append(f.contentCalls, in)
	return nil
}
func (f *fakeBolClient) GetCommission(ctx context.Context, ean string, unitPrice float64) (integrationsDomain.BolCommission, error) {
	if f.getCommissionFn != nil {
		return f.getCommissionFn(ctx, ean, unitPrice)
	}
	return integrationsDomain.BolCommission{EAN: ean}, nil
}
func (f *fakeBolClient) ListOrders(ctx context.Context, page int, status string) ([]integrationsDomain.BolOrder, error) {
	if f.listOrdersFn != nil {
		return f.listOrdersFn(ctx, page, status)
	}
	return nil, nil
}
func (f *fakeBolClient) GetOrder(ctx context.Context, orderID string) (*integrationsDomain.BolOrder, error) {
	if f.getOrderFn != nil {
		return f.getOrderFn(ctx, orderID)
	}
	return &integrationsDomain.BolOrder{OrderID: orderID}, nil
}

type fakeBolClientFactory struct{ client integrationsDomain.BolClient }

func (f *fakeBolClientFactory) NewClient(_, _ string) (integrationsDomain.BolClient, error) {
	return f.client, nil
}

type fakeBolOverrideRepo struct {
	override *channelsDomain.BolProductOverride
	upserted []channelsDomain.BolProductOverride
}

func (f *fakeBolOverrideRepo) FindByProductAndConnection(_ context.Context, _, _ string) (*channelsDomain.BolProductOverride, error) {
	return f.override, nil
}
func (f *fakeBolOverrideRepo) Upsert(_ context.Context, o *channelsDomain.BolProductOverride) error {
	o.ID = "override-1"
	f.upserted = append(f.upserted, *o)
	return nil
}

// fakeSKUByIDRepo backs GetBolCommissionUseCase's narrow skuFinder
// dependency (ISP) -- doesn't implement the full pmsDomain.SKURepository.
type fakeSKUByIDRepo struct {
	skus map[string]pmsDomain.SKU
}

func (f *fakeSKUByIDRepo) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	sku, ok := f.skus[id]
	if !ok {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return &sku, nil
}

func bolConn() integrationsDomain.Connection {
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "secret")
	if err != nil {
		panic(err)
	}
	return integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderBol,
		ConsumerKey: "client-id", ConsumerSecretEncrypted: encrypted,
		BolEconomicOperatorID: strPtr("op-1"),
	}
}

type pushHarness struct {
	connRepo     *stubConnectionRepo
	productRepo  *stubChannelsProductRepo
	variantRepo  *stubChannelsVariantRepo
	skuRepo      *stubChannelsSKURepo
	listingRepo  *stubListingRepo
	overrideRepo *fakeBolOverrideRepo
	syncLogRepo  *stubSyncLogRepo
	stockLevel   *stubStockLevelGetter
}

func newBolPushHarness(product pmsDomain.Product, variants []pmsDomain.Variant, skusByVariant map[string][]pmsDomain.SKU) *pushHarness {
	conn := bolConn()
	return &pushHarness{
		connRepo:     &stubConnectionRepo{conn: &conn},
		productRepo:  &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{product.ID: product}},
		variantRepo:  &stubChannelsVariantRepo{byProduct: map[string][]pmsDomain.Variant{product.ID: variants}},
		skuRepo:      &stubChannelsSKURepo{byVariant: skusByVariant},
		listingRepo:  &stubListingRepo{listByProduct: map[string][]channelsDomain.ProductChannelListing{}},
		overrideRepo: &fakeBolOverrideRepo{},
		syncLogRepo:  &stubSyncLogRepo{},
		stockLevel:   &stubStockLevelGetter{},
	}
}

func newBolUseCase(t *testing.T, h *pushHarness, client integrationsDomain.BolClient, mockTxCount int) *appchannels.PushProductToBolUseCase {
	t.Helper()
	return appchannels.NewPushProductToBolUseCase(
		newMockGormDB(t, mockTxCount), h.connRepo, &fakeBolClientFactory{client: client}, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.overrideRepo, h.syncLogRepo, h.stockLevel,
	)
}

func TestPushProductToBol_createsOfferForSingleSKUProduct(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	client := &fakeBolClient{createOfferFn: func(_ context.Context, in integrationsDomain.BolOfferInput) (string, error) {
		assert.Equal(t, "0630509892495", in.EAN)
		assert.Equal(t, "op-1", in.EconomicOperatorID)
		assert.Equal(t, "FBR", in.FulfilmentMethod)
		assert.Equal(t, "NEW", in.ConditionCategory)
		assert.Equal(t, "3-5d", in.DeliveryCode, "falls back to DefaultBolDeliveryCode when the connection hasn't set one")
		assert.InDelta(t, 19.99, in.UnitPrice, 0.001)
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	result, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "offer-1", result[0].ExternalID)
	assert.Equal(t, "bol", result[0].Provider)
	assert.Nil(t, result[0].VariantID)
}

func TestPushProductToBol_usesConnectionDeliveryCodeOverDefault(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	deliveryCode := "1-2d"
	h.connRepo.conn.BolDeliveryCode = &deliveryCode

	client := &fakeBolClient{createOfferFn: func(_ context.Context, in integrationsDomain.BolOfferInput) (string, error) {
		assert.Equal(t, "1-2d", in.DeliveryCode)
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)
}

func TestPushProductToBol_createsOfferPerVariantForVariableProduct(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: true}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}, {ID: "var-2", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{
		"var-1": {{ID: "sku-1", Barcode: "1111111111111", PriceMinorUnits: 1000}},
		"var-2": {{ID: "sku-2", Barcode: "2222222222222", PriceMinorUnits: 2000}},
	}
	h := newBolPushHarness(product, variants, skus)

	var eansSeen []string
	client := &fakeBolClient{createOfferFn: func(_ context.Context, in integrationsDomain.BolOfferInput) (string, error) {
		eansSeen = append(eansSeen, in.EAN)
		return "offer-" + in.EAN, nil
	}}

	uc := newBolUseCase(t, h, client, 5)
	result, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.ElementsMatch(t, []string{"1111111111111", "2222222222222"}, eansSeen)
	for _, l := range result {
		assert.NotNil(t, l.VariantID)
	}
}

func TestPushProductToBol_updatesExistingOfferInsteadOfCreating(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	h.listingRepo.listByProduct["prod-1"] = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "bol", ExternalID: "offer-existing", VariantID: nil},
	}

	client := &fakeBolClient{createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		t.Fatal("must not call CreateOffer when a listing already exists")
		return "", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	result, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.updateOfferCalls, 1)
	require.Len(t, result, 1)
	assert.Equal(t, "offer-existing", result[0].ExternalID)
}

func TestPushProductToBol_onUnknownEAN_submitsProductContentThenRetriesCreate(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	unknownEAN := errors.New("unknown ean")
	attempts := 0
	client := &fakeBolClient{unknownEANErr: unknownEAN, createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		attempts++
		if attempts == 1 {
			return "", unknownEAN
		}
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	result, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	require.Len(t, client.contentCalls, 1)
	assert.Equal(t, "0630509892495", client.contentCalls[0].EAN)
	require.Len(t, result, 1)
	assert.Equal(t, "offer-1", result[0].ExternalID)
}

func TestPushProductToBol_missingEAN_returnsError(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	client := &fakeBolClient{createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		t.Fatal("must not call CreateOffer with no EAN")
		return "", nil
	}}

	uc := newBolUseCase(t, h, client, 2)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, channelsDomain.ErrBolMissingEAN)
	require.Len(t, h.syncLogRepo.created, 1, "missing EAN must also show up in Sync History")
	assert.Equal(t, channelsDomain.SyncLogStatusError, h.syncLogRepo.created[0].Status)
}

func TestPushProductToBol_usesCustomPriceOverSKUDefaultPrice(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	customPrice := int64(2599)
	h.overrideRepo.override = &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1", CustomPriceMinorUnits: &customPrice,
	}

	client := &fakeBolClient{createOfferFn: func(_ context.Context, in integrationsDomain.BolOfferInput) (string, error) {
		assert.InDelta(t, 25.99, in.UnitPrice, 0.001)
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)
}

func TestPushProductToBol_recordsSyncLog(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	client := &fakeBolClient{createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	assert.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	assert.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	assert.Equal(t, "bol", entry.Provider)
	assert.Equal(t, "19.99", entry.Fields["price"])
}

// TestPushProductToBol_createOfferFailure_recordsFailedSyncLog is a
// regression test for a real prod bug: pushOne only ever called
// recordSyncLog on its success paths, so a CreateOffer failure (e.g. Bol's
// real 400 validation error) propagated all the way up as a Go error but
// never left any trace in Sync History -- from the UI's perspective, the
// push simply never happened. This is the failure-path counterpart to
// TestPushProductToBol_recordsSyncLog above.
func TestPushProductToBol_createOfferFailure_recordsFailedSyncLog(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	client := &fakeBolClient{createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		return "", errors.New(`bol: request failed (400): {"violations":[{"name":"condition.name","reason":"required"}]}`)
	}}

	uc := newBolUseCase(t, h, client, 2)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.Error(t, err)
	require.Len(t, h.syncLogRepo.created, 1, "a failed push must still be recorded so it shows up in Sync History")
	entry := h.syncLogRepo.created[0]
	assert.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	assert.Equal(t, channelsDomain.SyncLogStatusError, entry.Status)
	assert.Equal(t, "bol", entry.Provider)
	assert.Equal(t, "prod-1", entry.ProductID)
	assert.Equal(t, "conn-1", entry.ConnectionID)
	assert.Contains(t, entry.ErrorMessage, "condition.name")
	assert.Nil(t, entry.ListingID, "no ProductChannelListing was ever created for this failed first push")
}

func TestPushProductToBol_skipsBulkPoolSKU(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", UnitType: pmsDomain.SKUUnitTypeKilo}}}
	h := newBolPushHarness(product, variants, skus)
	client := &fakeBolClient{}

	uc := newBolUseCase(t, h, client, 1)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, channelsDomain.ErrCannotPushBulkPoolSKU)
}

func TestPushProductToBol_appliesOverrideEANAndFields(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0000000000000", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	h.overrideRepo.override = &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1",
		EANOverride: strPtr("9999999999999"), ConditionCategory: "REFURBISHED", FulfilmentMethod: "FBR",
	}

	client := &fakeBolClient{createOfferFn: func(_ context.Context, in integrationsDomain.BolOfferInput) (string, error) {
		assert.Equal(t, "9999999999999", in.EAN)
		assert.Equal(t, "REFURBISHED", in.ConditionCategory)
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)
}
