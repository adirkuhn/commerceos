package pms

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, pmsDomain.ErrProductNotFound),
		errors.Is(err, pmsDomain.ErrCategoryNotFound),
		errors.Is(err, pmsDomain.ErrVariantNotFound),
		errors.Is(err, pmsDomain.ErrSKUNotFound),
		errors.Is(err, pmsDomain.ErrMediaNotFound),
		errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound),
		errors.Is(err, pmsDomain.ErrScaleCodeAliasNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, pmsDomain.ErrDuplicateSlug),
		errors.Is(err, pmsDomain.ErrDuplicateSKUCode),
		errors.Is(err, pmsDomain.ErrDuplicateBarcode),
		errors.Is(err, pmsDomain.ErrDuplicateScaleCode),
		errors.Is(err, pmsDomain.ErrDuplicateBarcodeAliasCode),
		errors.Is(err, pmsDomain.ErrDuplicateScaleCodeAlias):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	case errors.Is(err, pmsDomain.ErrProductMergeForbidden):
		middleware.FailErr(c, http.StatusForbidden, "FORBIDDEN", err)
	case errors.Is(err, pmsDomain.ErrInvalidProductMerge):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, pmsDomain.ErrProductHasHistory):
		middleware.FailErr(c, http.StatusConflict, "PRODUCT_HAS_HISTORY", err)
	case errors.Is(err, pmsDomain.ErrVariantsNotEnabled),
		errors.Is(err, pmsDomain.ErrInvalidSlug),
		errors.Is(err, pmsDomain.ErrInvalidProductTranslation),
		errors.Is(err, pmsDomain.ErrInvalidStockLevel),
		errors.Is(err, pmsDomain.ErrInvalidLowStockThreshold),
		errors.Is(err, pmsDomain.ErrVirtualSKUCannotHaveDimensions),
		errors.Is(err, pmsDomain.ErrSalePriceExceedsPrice),
		errors.Is(err, pmsDomain.ErrSaleEndDateRequiresSalePrice),
		errors.Is(err, pmsDomain.ErrUnsupportedMediaType),
		errors.Is(err, pmsDomain.ErrMediaTargetRequired),
		errors.Is(err, pmsDomain.ErrFeaturedMediaProductMismatch),
		errors.Is(err, pmsDomain.ErrCaseOfSKURequiresUnitsPerCase),
		errors.Is(err, pmsDomain.ErrCaseOfSKUCannotBeSelf),
		errors.Is(err, pmsDomain.ErrCaseOfSKUCannotChain),
		errors.Is(err, pmsDomain.ErrUnitsPerCaseMustBePositive),
		errors.Is(err, pmsDomain.ErrPortionedFromSKUCannotBeSelf),
		errors.Is(err, pmsDomain.ErrPortionedFromSKUMustTrackWeight),
		errors.Is(err, pmsDomain.ErrBulkSKUCannotBePortioned),
		errors.Is(err, pmsDomain.ErrExchangeRateRequired),
		errors.Is(err, pmsDomain.ErrInvalidSKUPricingMode),
		errors.Is(err, pmsDomain.ErrPricePerKiloRequiredForPerWeight),
		errors.Is(err, pmsDomain.ErrPricePerKiloOnlyForPerWeight),
		errors.Is(err, pmsDomain.ErrInvalidPricePerKilo),
		errors.Is(err, pmsDomain.ErrInvalidCheckDigit),
		errors.Is(err, pmsDomain.ErrInvalidUnitsPerScan),
		errors.Is(err, pmsDomain.ErrBarcodeAliasCodeRequired),
		errors.Is(err, pmsDomain.ErrScaleCodeAliasCodeRequired),
		errors.Is(err, pmsDomain.ErrCostPriceUnknownForCorrection),
		errors.Is(err, pmsDomain.ErrNoStockOnHandForCorrection),
		errors.Is(err, fxDomain.ErrInvalidCurrencyCode):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("pms: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
