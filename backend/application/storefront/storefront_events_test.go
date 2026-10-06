package storefront_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

// Tokens are secrets: an event names the record by id, never by token.
func assertNoSecret(t *testing.T, e eventrecordtest.Event, secrets ...string) {
	t.Helper()
	raw := e.Payload
	for _, s := range secrets {
		assert.NotContains(t, toString(raw), s)
	}
}

func toString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestCreateCart_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{}
	uc := appstorefront.NewCreateCartUseCase(newMockGormDB(t, 1), channelRepo, cartRepo, outbox)

	_, err := uc.Execute(context.Background(), "channel-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.cart.created", e.Subject)
	assert.Equal(t, "cart-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_channel_id": "channel-1"}, e.Data())
	assertNoSecret(t, e, cartRepo.created.Token)
}

func TestUpdateCartItems_RecordsItemsUpdated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", SalesChannelID: "channel-1"},
	}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 500, 10)
	uc := appstorefront.NewUpdateCartItemsUseCase(
		newMockGormDB(t, 2), cartRepo, channelRepo, sku, variant, product, override,
		&fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, stockLevel, nil, &fakeCouponCurrencyAmountRepo{}, outbox,
	)

	_, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 3}})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.cart.items_updated", e.Subject)
	assert.Equal(t, "cart-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"sales_channel_id": "channel-1", "line_count": float64(1),
		"items": []any{map[string]any{"sku_id": "sku-1", "sku_code": "SKU-A", "quantity": float64(3)}},
	}, e.Data())
	assert.Equal(t, map[string][2]any{"items": {map[string]any{}, map[string]any{"SKU-A": float64(3)}}}, e.Changes())
	assertNoSecret(t, e, "token-1")
}

func TestApplyCoupon_RecordsApplied(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	couponRepo := &fakeCouponRepoForCheckout{byCode: map[string]couponsDomain.Coupon{"SUMMER10": redeemableCoupon()}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, couponRepo, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{}, outbox)

	_, err := uc.Execute(context.Background(), "token-1", "summer10")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.cart.coupon_applied", e.Subject)
	assert.Equal(t, map[string]any{"coupon_code": "SUMMER10", "discount_minor_units": float64(100)}, e.Data())
	assert.Equal(t, map[string][2]any{"coupon_code": {nil, "SUMMER10"}}, e.Changes())
}

func TestApplyCoupon_RecordsRemoved(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	cart := openCartWithOneItem()
	code := "SUMMER10"
	cart.CouponCode = &code
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": cart}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, &fakeCouponRepoForCheckout{}, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{}, outbox)

	_, err := uc.Execute(context.Background(), "token-1", "")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.cart.coupon_removed", e.Subject)
	assert.Equal(t, map[string][2]any{"coupon_code": {"SUMMER10", nil}}, e.Changes())
}

func TestCreateWishlist_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	wishlists := &fakeWishlistRepo{}
	uc := appstorefront.NewCreateWishlistUseCase(newMockGormDB(t, 1), channelRepo, wishlists, outbox)

	_, err := uc.Execute(context.Background(), "channel-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.wishlist.created", e.Subject)
	assert.Equal(t, "wishlist-1", e.Payload["entity_id"])
	assertNoSecret(t, e, wishlists.created.Token)
}

func TestToggleWishlistItem_RecordsAddedAndRemoved(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	wishlists := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", SalesChannelID: "channel-1"},
	}}
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	uc := appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), wishlists, channelRepo, overrideRepo, productRepo, outbox)

	_, err := uc.Execute(context.Background(), "token-1", "product-1")
	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.wishlist.item_added", e.Subject)
	assert.Equal(t, map[string]any{"sales_channel_id": "channel-1", "product_id": "product-1"}, e.Data())

	removed := &eventrecordtest.Outbox{}
	saved := &fakeWishlistRepo{byToken: map[string]storefrontDomain.Wishlist{
		"token-1": {ID: "wishlist-1", Token: "token-1", SalesChannelID: "channel-1", ProductIDs: []string{"product-1"}},
	}}
	_, err = appstorefront.NewToggleWishlistItemUseCase(newMockGormDB(t, 1), saved, &fakeChannelRepo{}, &fakeOverrideRepo{}, &fakeProductRepo{}, removed).
		Execute(context.Background(), "token-1", "product-1")
	require.NoError(t, err)
	assert.Equal(t, "storefront.wishlist.item_removed", onlyEvent(t, removed).Subject)
}

func TestSubmitReview_RecordsSubmittedWithoutPersonalData(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	channelRepo, productRepo, overrideRepo := activeEnrolledProduct()
	uc := appstorefront.NewSubmitReviewUseCase(newMockGormDB(t, 1), channelRepo, overrideRepo, productRepo, &fakeReviewRepo{verified: true}, outbox)

	_, err := uc.Execute(context.Background(), appstorefront.SubmitReviewInput{
		SalesChannelID: "channel-1", ProductID: "product-1",
		CustomerName: "Ada", CustomerEmail: "ada@example.com", Rating: 5, Title: "Great", Body: "Loved it",
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.review.submitted", e.Subject)
	assert.Equal(t, "review-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_channel_id": "channel-1", "product_id": "product-1", "rating": float64(5), "verified": true}, e.Data())
	assertNoSecret(t, e, "ada@example.com", "Ada")
}

func TestRequestMagicLink_RecordsRequestedWithoutEmailOrToken(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	tokenRepo := &fakeMagicLinkTokenRepo{}
	uc := appstorefront.NewRequestMagicLinkUseCase(newMockGormDB(t, 2),
		&fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": headlessChannel("channel-1", "https://shop.example.com/auth/verify")}},
		&fakeCustomerRepo{byEmail: map[string]customersDomain.Customer{"shopper@example.com": {ID: "customer-1"}}},
		&fakeCustomerCreator{}, tokenRepo, &fakeEmailClient{}, hostedStorefronts, outbox)

	err := uc.Execute(context.Background(), appstorefront.RequestMagicLinkInput{SalesChannelID: "channel-1", Email: "shopper@example.com"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.magic_link.requested", e.Subject)
	assert.Equal(t, "magic-link-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_channel_id": "channel-1", "customer_id": "customer-1"}, e.Data())
	assertNoSecret(t, e, "shopper@example.com", tokenRepo.created.Token)
}

func TestVerifyMagicLink_RecordsSessionStarted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	tokenRepo := &fakeMagicLinkTokenRepo{byToken: map[string]storefrontDomain.MagicLinkToken{"LINK123": redeemableMagicLink()}}
	uc := appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1), tokenRepo, &fakeCustomerSessionRepo{}, outbox)

	session, err := uc.Execute(context.Background(), "channel-1", "LINK123")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.customer_session.started", e.Subject)
	assert.Equal(t, "session-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_channel_id": "channel-1", "customer_id": "customer-1", "magic_link_id": "magic-link-1"}, e.Data())
	assertNoSecret(t, e, "LINK123", session.Token)
}

func TestLogout_RecordsSessionEnded(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	sessions := &fakeCustomerSessionRepo{byToken: map[string]storefrontDomain.CustomerSession{
		"SESSION123": {ID: "session-7", SalesChannelID: "channel-1", CustomerID: "customer-1", Token: "SESSION123"},
	}}
	uc := appstorefront.NewLogoutUseCase(newMockGormDB(t, 1), sessions, outbox)

	require.NoError(t, uc.Execute(context.Background(), "SESSION123"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.customer_session.ended", e.Subject)
	assert.Equal(t, "session-7", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_channel_id": "channel-1", "customer_id": "customer-1"}, e.Data())
	assertNoSecret(t, e, "SESSION123")
}

// Signing out of a session that's already gone changes nothing.
func TestLogout_UnknownSessionRecordsNothing(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	require.NoError(t, appstorefront.NewLogoutUseCase(newMockGormDB(t, 1), &fakeCustomerSessionRepo{}, outbox).Execute(context.Background(), "gone"))
	assert.Empty(t, outbox.Events)
}

type fakeConnectionRepoForLink struct {
	integrationsDomain.ConnectionRepository
	conn integrationsDomain.Connection
}

func (f *fakeConnectionRepoForLink) FindByID(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
	c := f.conn
	return &c, nil
}

type fakeLinkClient struct {
	link integrationsDomain.PaymentLink
}

func (f *fakeLinkClient) CreatePaymentLink(_ context.Context, _ integrationsDomain.PaymentLinkRequest) (integrationsDomain.PaymentLink, error) {
	return f.link, nil
}

func (f *fakeLinkClient) Refund(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return "", nil
}

type fakeLinkFactory struct{ client *fakeLinkClient }

func (f *fakeLinkFactory) NewClient(_ string) (integrationsDomain.PaymentLinkClient, error) {
	return f.client, nil
}

const linkSecret = "test-encryption-secret"

func TestGeneratePaymentLink_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	enc, err := crypto.Encrypt(linkSecret, "sk_test")
	require.NoError(t, err)
	connID := "conn-1"
	channel := internalChannel("channel-1")
	channel.PaymentConnectionID = &connID
	uc := appstorefront.NewGeneratePaymentLinkUseCase(newMockGormDB(t, 2),
		&fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": channel}},
		&fakeConnectionRepoForLink{conn: integrationsDomain.Connection{ID: connID, Provider: integrationsDomain.ProviderStripe, ConsumerSecretEncrypted: enc}},
		&fakeLinkFactory{client: &fakeLinkClient{link: integrationsDomain.PaymentLink{URL: "https://pay/1", ExternalID: "cs_1"}}},
		&fakeLinkFactory{}, linkSecret, "https://app.example.com", outbox)

	_, ok, err := uc.Execute(context.Background(), "channel-1", 1999, "EUR", "Order #1001", "1001")

	require.NoError(t, err)
	require.True(t, ok)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "storefront.payment_link.created", e.Subject)
	assert.Equal(t, "cs_1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"sales_channel_id": "channel-1", "provider": "stripe", "reference": "1001",
		"amount_minor_units": float64(1999), "currency_code": "EUR",
	}, e.Data())
}

func TestStorefrontUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()
	channels := func() *fakeChannelRepo {
		return &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	}

	_, err := appstorefront.NewCreateCartUseCase(newMockGormDB(t, 1), channels(), &fakeCartRepo{}, failing).Execute(ctx, "channel-1")
	assert.ErrorIs(t, err, boom, "cart")
	_, err = appstorefront.NewCreateWishlistUseCase(newMockGormDB(t, 1), channels(), &fakeWishlistRepo{}, failing).Execute(ctx, "channel-1")
	assert.ErrorIs(t, err, boom, "wishlist")
	_, err = appstorefront.NewVerifyMagicLinkUseCase(newMockGormDB(t, 1),
		&fakeMagicLinkTokenRepo{byToken: map[string]storefrontDomain.MagicLinkToken{"LINK123": redeemableMagicLink()}}, &fakeCustomerSessionRepo{}, failing).
		Execute(ctx, "channel-1", "LINK123")
	assert.ErrorIs(t, err, boom, "verify")
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	_, err = appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo,
		&fakeCouponRepoForCheckout{byCode: map[string]couponsDomain.Coupon{"SUMMER10": redeemableCoupon()}}, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{}, failing).
		Execute(ctx, "token-1", "SUMMER10")
	assert.ErrorIs(t, err, boom, "coupon")
}
