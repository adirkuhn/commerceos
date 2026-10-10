package storefront

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// shopperError is how one domain error reaches the storefront: an HTTP
// status and a stable code the storefront translates into the shop's
// language (shop.json errors.<code>). The message is the English sentence
// without internal call-site prefixes, for logs and API clients.
type shopperError struct {
	err    error
	status int
	code   string
}

var shopperErrors = []shopperError{
	{appreports.ErrCartEmpty, http.StatusUnprocessableEntity, "CART_EMPTY"},
	{storefrontDomain.ErrCartEmpty, http.StatusUnprocessableEntity, "CART_EMPTY"},
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

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	for _, se := range shopperErrors {
		if errors.Is(err, se.err) {
			middleware.Fail(c, se.status, se.code, se.err.Error())
			return err
		}
	}
	log.Printf("storefront: unhandled error: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	return err
}
