package procurement

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, procurementDomain.ErrPurchaseOrderNotFound), errors.Is(err, procurementDomain.ErrLineItemNotFound),
		errors.Is(err, procurementDomain.ErrDocumentNotFound), errors.Is(err, procurementDomain.ErrReceivingRecordNotFound),
		errors.Is(err, procurementDomain.ErrBookingNotFound), errors.Is(err, procurementDomain.ErrParsingJobNotFound),
		errors.Is(err, procurementDomain.ErrBookingScanNotFound), errors.Is(err, procurementDomain.ErrDiscrepancyNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, procurementDomain.ErrInvalidQuantity), errors.Is(err, procurementDomain.ErrUnsupportedFileType),
		errors.Is(err, procurementDomain.ErrMissingRequiredColumns), errors.Is(err, procurementDomain.ErrPDFHasNoExtractableText),
		errors.Is(err, procurementDomain.ErrAIParsingNotConfigured), errors.Is(err, procurementDomain.ErrLineItemAlreadyLinked),
		errors.Is(err, procurementDomain.ErrNoLineItemsExtracted),
		errors.Is(err, procurementDomain.ErrBookingNotActive), errors.Is(err, procurementDomain.ErrWarehouseRequired),
		errors.Is(err, procurementDomain.ErrScanCodeOrSKURequired), errors.Is(err, procurementDomain.ErrInvalidBookingScanActionType),
		errors.Is(err, procurementDomain.ErrPendingBatchNotFound),
		errors.Is(err, procurementDomain.ErrDiscrepancyMismatch), errors.Is(err, procurementDomain.ErrDiscrepancyAlreadyAcknowledged),
		errors.Is(err, procurementDomain.ErrInvalidDamagedQuantity),
		errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, invDomain.ErrInsufficientStock),
		errors.Is(err, pmsDomain.ErrSKUNotFound), errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound),
		errors.Is(err, pmsDomain.ErrDuplicateSlug), errors.Is(err, pmsDomain.ErrDuplicateSKUCode),
		errors.Is(err, pmsDomain.ErrWeightRequiredForPerWeightSKU), errors.Is(err, pmsDomain.ErrInvalidCheckDigit):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("procurement: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
