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

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, appreports.ErrCartEmpty),
		errors.Is(err, storefrontDomain.ErrCartEmpty),
		errors.Is(err, storefrontDomain.ErrInsufficientStock),
		errors.Is(err, storefrontDomain.ErrProductNotAvailable),
		errors.Is(err, storefrontDomain.ErrNoInternalChannel),
		errors.Is(err, couponsDomain.ErrCouponNotRedeemable),
		errors.Is(err, couponsDomain.ErrCouponMinSubtotalNotMet),
		errors.Is(err, reviewsDomain.ErrCustomerNameRequired),
		errors.Is(err, reviewsDomain.ErrCustomerEmailRequired),
		errors.Is(err, reviewsDomain.ErrInvalidRating),
		errors.Is(err, reviewsDomain.ErrReviewBodyRequired),
		errors.Is(err, storefrontDomain.ErrInvalidOrExpiredMagicLink),
		errors.Is(err, storefrontDomain.ErrCardPaymentNotSupported),
		errors.Is(err, storefrontDomain.ErrCardPaymentMismatch):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, storefrontDomain.ErrCartNotFound), errors.Is(err, storefrontDomain.ErrOrderNotFound),
		errors.Is(err, storefrontDomain.ErrWishlistNotFound), errors.Is(err, storefrontDomain.ErrSessionNotFound),
		errors.Is(err, couponsDomain.ErrCouponNotFound),
		errors.Is(err, channelsDomain.ErrSalesChannelNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, storefrontDomain.ErrCartAlreadyCheckedOut):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	default:
		log.Printf("storefront: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
