package storefront_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	storefronthandler "github.com/adirkuhn/commerceos/infrastructure/handlers/storefront"
)

type subscribeF func(context.Context, appnewsletter.SubscribeInput) error

func (f subscribeF) Execute(ctx context.Context, in appnewsletter.SubscribeInput) error {
	return f(ctx, in)
}

type tokenF func(context.Context, string) error

func (f tokenF) Execute(ctx context.Context, tok string) error { return f(ctx, tok) }

func TestNewsletter_storefrontEndpoints(t *testing.T) {
	var got appnewsletter.SubscribeInput
	var gotToken string
	var err error
	news := storefronthandler.NewsletterUseCases{
		Subscribe:   subscribeF(func(_ context.Context, in appnewsletter.SubscribeInput) error { got = in; return err }),
		Confirm:     tokenF(func(_ context.Context, tok string) error { gotToken = tok; return err }),
		Unsubscribe: tokenF(func(_ context.Context, tok string) error { gotToken = tok; return err }),
	}
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	r := newTestRouter(newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{},
		&stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, news))

	w := doRequest(r, http.MethodPost, "/api/v1/storefront/v1/newsletter/subscriptions", `{"email":"a@x.io","consent":true}`, "key")
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"status":"check_your_email"`)
	assert.Equal(t, appnewsletter.SubscribeInput{SalesChannelID: "channel-1", Email: "a@x.io", Consent: true}, got)

	w = doRequest(r, http.MethodPost, "/api/v1/storefront/v1/newsletter/confirmations", `{"token":"t1"}`, "key")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "t1", gotToken)
	assert.Contains(t, w.Body.String(), `"status":"subscribed"`)

	w = doRequest(r, http.MethodPost, "/api/v1/storefront/v1/newsletter/unsubscriptions", `{"token":"u1"}`, "key")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"unsubscribed"`)

	for _, e := range []error{newsletter.ErrInvalidEmail, newsletter.ErrConsentRequired, newsletter.ErrInvalidToken} {
		err = e
		w = doRequest(r, http.MethodPost, "/api/v1/storefront/v1/newsletter/subscriptions", `{"email":"x","consent":true}`, "key")
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, e.Error())
	}
	assert.Equal(t, http.StatusUnauthorized, doRequest(r, http.MethodPost, "/api/v1/storefront/v1/newsletter/subscriptions", `{}`, "").Code)
}
