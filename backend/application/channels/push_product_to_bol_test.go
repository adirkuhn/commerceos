package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// --- Bol-specific fakes (BolClient/BolClientFactory/BolProductOverrideRepository have no other test double in this package yet) ---

type fakeBolClient struct {
	createOfferFn           func(ctx context.Context, in integrationsDomain.BolOfferInput) (string, error)
	updateOfferCalls        []integrationsDomain.BolOfferInput
	contentCalls            []integrationsDomain.BolProductContentInput
	unknownEANErr           error
	getCommissionFn         func(ctx context.Context, ean string, unitPrice float64) (integrationsDomain.BolCommission, error)
	listOrdersFn            func(ctx context.Context, page int, status string) ([]integrationsDomain.BolOrder, error)
	getOrderFn              func(ctx context.Context, orderID string) (*integrationsDomain.BolOrder, error)
	listEconomicOperatorsFn func(ctx context.Context) ([]integrationsDomain.BolEconomicOperator, error)
	findOfferByEANFn        func(ctx context.Context, ean string) (string, bool, error)
	notForSaleReasonsFn     func(ctx context.Context, offerID string) ([]integrationsDomain.BolNotForSaleReason, error)
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
func (f *fakeBolClient) FindOfferByEAN(ctx context.Context, ean string) (string, bool, error) {
	if f.findOfferByEANFn != nil {
		return f.findOfferByEANFn(ctx, ean)
	}
	return "", false, nil
}
func (f *fakeBolClient) ListEconomicOperators(ctx context.Context) ([]integrationsDomain.BolEconomicOperator, error) {
	if f.listEconomicOperatorsFn != nil {
		return f.listEconomicOperatorsFn(ctx)
	}
	return nil, nil
}
func (f *fakeBolClient) GetNotForSaleReasons(ctx context.Context, offerID string) ([]integrationsDomain.BolNotForSaleReason, error) {
	if f.notForSaleReasonsFn != nil {
		return f.notForSaleReasonsFn(ctx, offerID)
	}
	return nil, nil
}

// fakeNotifier is the default notifier newBolUseCase wires in -- tests that
// don't care about notification content get a no-op success; tests that do
// (see TestPushProductToBol_notForSale_*) construct their own.
type fakeNotifier struct {
	calls []appnotifications.CreateNotificationInput
	err   error
}

func (f *fakeNotifier) Execute(_ context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	return &notificationsDomain.Notification{Type: in.Type, Title: in.Title, Message: in.Message, Link: in.Link}, nil
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

type fakeBolContentRepo struct {
	content  *channelsDomain.BolProductContent
	upserted []channelsDomain.BolProductContent
}

func (f *fakeBolContentRepo) FindByProductAndConnection(_ context.Context, _, _ string) (*channelsDomain.BolProductContent, error) {
	return f.content, nil
}
func (f *fakeBolContentRepo) Upsert(_ context.Context, c *channelsDomain.BolProductContent) error {
	c.ID = "content-1"
	f.upserted = append(f.upserted, *c)
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
	contentRepo  *fakeBolContentRepo
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
		contentRepo:  &fakeBolContentRepo{},
		syncLogRepo:  &stubSyncLogRepo{},
		stockLevel:   &stubStockLevelGetter{},
	}
}

func newBolUseCase(t *testing.T, h *pushHarness, client integrationsDomain.BolClient, mockTxCount int) *appchannels.PushProductToBolUseCase {
	t.Helper()
	return appchannels.NewPushProductToBolUseCase(
		newMockGormDB(t, mockTxCount), h.connRepo, &fakeBolClientFactory{client: client}, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.overrideRepo, h.contentRepo, h.syncLogRepo, h.stockLevel,
		&fakeNotifier{},
	)
}

func newBolUseCaseWithNotifier(t *testing.T, h *pushHarness, client integrationsDomain.BolClient, mockTxCount int, notifier *fakeNotifier) *appchannels.PushProductToBolUseCase {
	t.Helper()
	return appchannels.NewPushProductToBolUseCase(
		newMockGormDB(t, mockTxCount), h.connRepo, &fakeBolClientFactory{client: client}, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.overrideRepo, h.contentRepo, h.syncLogRepo, h.stockLevel,
		notifier,
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

// TestPushProductToBol_updatingExistingOffer_alsoSubmitsProductContent is a
// regression test for a real bug: SubmitProductContent only ever fired on
// the unknown-EAN create-retry path, so a product whose offer already
// exists (the common case -- every push after the first) took the UPDATE
// branch and never submitted product content at all, no matter how many
// fields the merchant saved on the Bol tab's "Product content" section.
// Filling in those fields and clicking Save/Push looked like it worked
// (the offer update succeeded) but Bol's catalog data never changed.
func TestPushProductToBol_updatingExistingOffer_alsoSubmitsProductContent(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	h.listingRepo.listByProduct["prod-1"] = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "bol", ExternalID: "offer-existing", VariantID: nil},
	}
	brand := "Hipoglós"
	h.contentRepo.content = &channelsDomain.BolProductContent{ProductID: "prod-1", ConnectionID: "conn-1", Brand: &brand}

	client := &fakeBolClient{}

	uc := newBolUseCase(t, h, client, 3)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.updateOfferCalls, 1)
	require.Len(t, client.contentCalls, 1, "product content must be submitted even when the offer already exists and only needs an update")
	assert.Equal(t, "Hipoglós", client.contentCalls[0].Attributes["Brand"])
}

// TestPushProductToBol_matchesExistingBolOfferByEAN_updatesInsteadOfCreating
// is a regression test for "no link yet between our product and any Bol
// offer": with no local ProductChannelListing row, the old code always
// assumed no Bol offer existed either and blindly POSTed a create -- which
// 409 Conflicts if an offer for this EAN already exists on Bol (created
// before this integration, or orphaned by a lost listing row). Now it
// checks FindOfferByEAN first and links+updates the existing offer.
func TestPushProductToBol_matchesExistingBolOfferByEAN_updatesInsteadOfCreating(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	client := &fakeBolClient{
		findOfferByEANFn: func(_ context.Context, ean string) (string, bool, error) {
			assert.Equal(t, "0630509892495", ean)
			return "offer-preexisting", true, nil
		},
		createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
			t.Fatal("must not call CreateOffer when FindOfferByEAN already found a match")
			return "", nil
		},
	}

	uc := newBolUseCase(t, h, client, 3)
	result, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.updateOfferCalls, 1)
	require.Len(t, result, 1)
	assert.Equal(t, "offer-preexisting", result[0].ExternalID, "the local listing must link to the matched Bol offer, not a newly created one")
}

func TestPushProductToBol_findOfferByEANFailure_recordsFailedSyncLog(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)

	client := &fakeBolClient{
		findOfferByEANFn: func(_ context.Context, _ string) (string, bool, error) {
			return "", false, errors.New("bol: request failed (500): internal error")
		},
	}

	uc := newBolUseCase(t, h, client, 2)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.Error(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	assert.Equal(t, channelsDomain.SyncLogStatusError, h.syncLogRepo.created[0].Status)
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

// TestPushProductToBol_onUnknownEAN_includesProductContentAttributes
// proves BolProductContent fields the merchant filled in under the
// product's Bol tab actually reach SubmitProductContent -- without this,
// the fields exist in the database and the UI but never make it into the
// payload Bol actually receives.
func TestPushProductToBol_onUnknownEAN_includesProductContentAttributes(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	brand := "Hipoglós"
	h.contentRepo.content = &channelsDomain.BolProductContent{ProductID: "prod-1", ConnectionID: "conn-1", Brand: &brand}

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
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.contentCalls, 1)
	assert.Equal(t, "Hipoglós", client.contentCalls[0].Attributes["Brand"])
}

// TestPushProductToBol_manufacturerFields_useBolsRealAttributeIds is a
// regression test: these three fields used to be keyed by their Dutch UI
// label ("Fabrikant Naam" etc), which Bol's API silently ignores -- no
// error, the POST 202s, the fields just never appear on bol.com. Confirmed
// against Bol's real, published data model
// (productdatamodel.s-bol.com/v10/datamodel_v10_en.json) that the real ids
// are the English strings asserted below.
func TestPushProductToBol_manufacturerFields_useBolsRealAttributeIds(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	name, address, contact := "Acme BV", "Main St 1, Amsterdam", "support@acme.example"
	h.contentRepo.content = &channelsDomain.BolProductContent{
		ProductID: "prod-1", ConnectionID: "conn-1",
		ManufacturerName: &name, ManufacturerAddress: &address, ManufacturerContact: &contact,
	}

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
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.contentCalls, 1)
	attrs := client.contentCalls[0].Attributes
	assert.Equal(t, "Acme BV", attrs["Manufacturer Name"])
	assert.Equal(t, "Main St 1, Amsterdam", attrs["Manufacturer Address"])
	assert.Equal(t, "support@acme.example", attrs["Manufacturer Email"])
	assert.NotContains(t, attrs, "Fabrikant Naam")
	assert.NotContains(t, attrs, "Fabrikant Adres")
	assert.NotContains(t, attrs, "E-mailadres of online contactformulier van de fabrikant")
}

// TestPushProductToBol_numberAttributes_carryBolsUnitId is a regression
// test for the second class of bug found alongside the wrong-id one: Bol's
// AttributeValue needs value and unitId as sibling fields for number-typed
// attributes, not two separate free-text attributes (which used to be sent
// as "Inhoud"/"Inhoud eenheid" -- both wrong ids AND wrong shape). Content
// amount+unit now merge into one "Capacity" attribute with a translated
// unitId; package width (always millimeters in this domain, no user-typed
// unit to translate) gets a fixed unitId.
func TestPushProductToBol_numberAttributes_carryBolsUnitId(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	amount, unit := "250", "ml"
	width := int32(80)
	h.contentRepo.content = &channelsDomain.BolProductContent{
		ProductID: "prod-1", ConnectionID: "conn-1",
		ContentAmount: &amount, ContentUnit: &unit, PackageWidthMM: &width,
	}

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
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.contentCalls, 1)
	call := client.contentCalls[0]
	assert.Equal(t, "250", call.Attributes["Capacity"])
	assert.Equal(t, "unece.unit.MLT", call.AttributeUnits["Capacity"])
	assert.Equal(t, "80", call.Attributes["Width"])
	assert.Equal(t, "unece.unit.MMT", call.AttributeUnits["Width"])
	assert.NotContains(t, call.Attributes, "Inhoud")
	assert.NotContains(t, call.Attributes, "Inhoud eenheid")
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

// TestPushProductToBol_contentOnlyChange_stillWritesASyncLogEntry is a
// regression test for a real prod bug: recordSyncLog's dedup (see its doc
// comment -- a successful sync identical to the last logged entry writes
// no row) diffed only bolOfferToFields (price/stock/ean/etc), never
// product-content attributes. A content-only save (e.g. filling in
// Manufacturer Name, no price/stock change) genuinely reached Bol and
// succeeded, but looked byte-for-byte identical to the previous log entry
// and got silently skipped -- from Sync History's perspective the sync
// never happened (caught 2026-08-25: content fixes deployed and pushed
// with no error, /sync-log showed nothing new).
func TestPushProductToBol_contentOnlyChange_stillWritesASyncLogEntry(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	client := &fakeBolClient{createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		return "offer-1", nil
	}}
	uc := newBolUseCase(t, h, client, 6) // two Execute calls, 3 tenant.WithTenant cycles each

	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1, "first push always logs, nothing to diff against yet")

	name := "Acme BV"
	h.contentRepo.content = &channelsDomain.BolProductContent{ProductID: "prod-1", ConnectionID: "conn-1", ManufacturerName: &name}

	_, err = uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 2, "content-only change (no price/stock movement) must still produce a new Sync History row")
	assert.Equal(t, "Acme BV", h.syncLogRepo.created[1].Fields["Manufacturer Name"])
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

// TestPushProductToBol_nameOverride_winsOverProductNameForBolTitleOnly
// proves the Bol tab's name override reaches Bol's Product Content API
// Title, Bol-only -- pms.Product.Name itself is left untouched (nothing
// else in this codebase reads Title from BolProductOverride), so Woo and
// every other channel keep syncing the real product name.
func TestPushProductToBol_nameOverride_winsOverProductNameForBolTitleOnly(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Real Product Name", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	// ReferenceOverride forces contentAttrs non-empty (unconditional
	// "Internal Reference" entry) so SubmitProductContent actually fires --
	// with no content/reference/gpc configured at all, this push would take
	// the CreateOffer-succeeds-immediately path and never call it.
	name, reference := "Custom Bol Title", "ref-1"
	h.overrideRepo.override = &channelsDomain.BolProductOverride{
		ProductID: "prod-1", ConnectionID: "conn-1", NameOverride: &name, ReferenceOverride: &reference,
	}
	client := &fakeBolClient{createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
		return "offer-1", nil
	}}

	uc := newBolUseCase(t, h, client, 3)
	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, client.contentCalls, 1)
	assert.Equal(t, "Custom Bol Title", client.contentCalls[0].Title)
	assert.Equal(t, "Real Product Name", product.Name, "the product's own name is never mutated by a Bol-only override")
}

// TestPushProductToBol_notForSale_savesNoteAndNotifies is a regression test
// for a real prod symptom: a product's Bol offer stayed "not for sale"
// (Bol's own seller portal message: "mandatory product information is
// missing") with zero error anywhere in this codebase -- every push
// 200/202'd cleanly. Bol only ever reports this via a separate diagnostic
// call (GET .../not-for-sale-reasons), which nothing called before this
// fix. Checked after every successful push; a non-empty result is both
// persisted on the listing (so the Bol tab can show it) and raises a
// notification linking back to the product.
func TestPushProductToBol_notForSale_savesNoteAndNotifies(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	client := &fakeBolClient{
		createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
			return "offer-1", nil
		},
		notForSaleReasonsFn: func(_ context.Context, offerID string) ([]integrationsDomain.BolNotForSaleReason, error) {
			assert.Equal(t, "offer-1", offerID)
			return []integrationsDomain.BolNotForSaleReason{{
				CountryCode: "NL", Category: "CONTENT", Code: "4004-2",
				Description:       "For this product essential product information is missing",
				MissingAttributes: []string{"Manufacturer Name"},
			}}, nil
		},
	}
	notifier := &fakeNotifier{}

	uc := newBolUseCaseWithNotifier(t, h, client, 3, notifier)
	result, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Contains(t, result[0].NotForSaleNote, "essential product information is missing")
	assert.Contains(t, result[0].NotForSaleNote, "Manufacturer Name")

	require.Len(t, notifier.calls, 1)
	assert.Equal(t, notificationsDomain.TypeBolNotForSale, notifier.calls[0].Type)
	assert.Equal(t, "/products?id=prod-1", notifier.calls[0].Link)
	assert.Contains(t, notifier.calls[0].Message, "Manufacturer Name")
}

// TestPushProductToBol_notForSale_doesNotRenotifyOnUnchangedReason proves a
// scheduled resync (which re-pushes on a timer, not just on merchant
// action) doesn't spam a notification every cycle for a problem the
// merchant already knows about -- only a change in the reason re-notifies.
func TestPushProductToBol_notForSale_doesNotRenotifyOnUnchangedReason(t *testing.T) {
	product := pmsDomain.Product{ID: "prod-1", Name: "Widget", HasVariants: false}
	variants := []pmsDomain.Variant{{ID: "var-1", ProductID: "prod-1"}}
	skus := map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", Barcode: "0630509892495", PriceMinorUnits: 1999}}}
	h := newBolPushHarness(product, variants, skus)
	client := &fakeBolClient{
		createOfferFn: func(_ context.Context, _ integrationsDomain.BolOfferInput) (string, error) {
			return "offer-1", nil
		},
		notForSaleReasonsFn: func(context.Context, string) ([]integrationsDomain.BolNotForSaleReason, error) {
			return []integrationsDomain.BolNotForSaleReason{{
				CountryCode: "NL", Category: "CONTENT", Code: "4004-2", Description: "still missing info",
			}}, nil
		},
	}
	notifier := &fakeNotifier{}
	uc := newBolUseCaseWithNotifier(t, h, client, 6, notifier)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)
	_, err = uc.Execute(context.Background(), appchannels.PushProductToBolInput{ProductID: "prod-1", ConnectionID: "conn-1"})
	require.NoError(t, err)

	assert.Len(t, notifier.calls, 1, "same reason on the second push must not fire a second notification")
}
