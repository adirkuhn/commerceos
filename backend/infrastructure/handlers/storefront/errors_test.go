package storefront

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func failWith(t *testing.T, err error) (int, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	require.Error(t, mapError(c, err))
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body.Error.Code, body.Error.Message
}

func TestMapError_givesEveryShopperErrorItsOwnCode(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{storefrontDomain.ErrCartEmpty, http.StatusUnprocessableEntity, "CART_EMPTY"},
		{appreports.ErrCartEmpty, http.StatusUnprocessableEntity, "CART_EMPTY"},
		{storefrontDomain.ErrInsufficientStock, http.StatusUnprocessableEntity, "INSUFFICIENT_STOCK"},
		{storefrontDomain.ErrProductNotAvailable, http.StatusUnprocessableEntity, "PRODUCT_NOT_AVAILABLE"},
		{storefrontDomain.ErrNoInternalChannel, http.StatusUnprocessableEntity, "STORE_NOT_AVAILABLE"},
		{storefrontDomain.ErrChannelWarehouseNotConfigured, http.StatusUnprocessableEntity, "STORE_NOT_AVAILABLE"},
		{couponsDomain.ErrCouponNotRedeemable, http.StatusUnprocessableEntity, "COUPON_NOT_REDEEMABLE"},
		{couponsDomain.ErrCouponMinSubtotalNotMet, http.StatusUnprocessableEntity, "COUPON_MIN_SUBTOTAL_NOT_MET"},
		{reviewsDomain.ErrCustomerNameRequired, http.StatusUnprocessableEntity, "REVIEW_NAME_REQUIRED"},
		{reviewsDomain.ErrCustomerEmailRequired, http.StatusUnprocessableEntity, "REVIEW_EMAIL_REQUIRED"},
		{reviewsDomain.ErrInvalidRating, http.StatusUnprocessableEntity, "REVIEW_INVALID_RATING"},
		{reviewsDomain.ErrReviewBodyRequired, http.StatusUnprocessableEntity, "REVIEW_BODY_REQUIRED"},
		{storefrontDomain.ErrInvalidOrExpiredMagicLink, http.StatusUnprocessableEntity, "SIGN_IN_LINK_INVALID"},
		{storefrontDomain.ErrCardPaymentNotSupported, http.StatusUnprocessableEntity, "CARD_PAYMENT_NOT_SUPPORTED"},
		{storefrontDomain.ErrCardPaymentMismatch, http.StatusUnprocessableEntity, "CARD_PAYMENT_MISMATCH"},
		{channelsDomain.ErrSignInNotConfigured, http.StatusUnprocessableEntity, "SIGN_IN_NOT_CONFIGURED"},
		{storefrontDomain.ErrCartNotFound, http.StatusNotFound, "CART_NOT_FOUND"},
		{storefrontDomain.ErrOrderNotFound, http.StatusNotFound, "ORDER_NOT_FOUND"},
		{storefrontDomain.ErrWishlistNotFound, http.StatusNotFound, "WISHLIST_NOT_FOUND"},
		{storefrontDomain.ErrSessionNotFound, http.StatusNotFound, "SESSION_NOT_FOUND"},
		{couponsDomain.ErrCouponNotFound, http.StatusNotFound, "COUPON_NOT_FOUND"},
		{channelsDomain.ErrSalesChannelNotFound, http.StatusNotFound, "STORE_NOT_AVAILABLE"},
		{storefrontDomain.ErrCartAlreadyCheckedOut, http.StatusConflict, "CART_ALREADY_CHECKED_OUT"},
	}
	for _, tc := range cases {
		wrapped := fmt.Errorf("checkoutCart: %w", tc.err)
		status, code, message := failWith(t, wrapped)
		assert.Equal(t, tc.status, status, tc.err.Error())
		assert.Equal(t, tc.code, code, tc.err.Error())
		assert.Equal(t, tc.err.Error(), message, "message is the plain sentence, no internal prefix")
	}
}

func TestMapError_unknownIsInternal(t *testing.T) {
	status, code, message := failWith(t, errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "INTERNAL", code)
	assert.Equal(t, "internal server error", message)
}
