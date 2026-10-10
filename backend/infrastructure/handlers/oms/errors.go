package oms

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, omsDomain.ErrSalesOrderNotFound), errors.Is(err, omsDomain.ErrSalesOrderLineItemNotFound),
		errors.Is(err, pmsDomain.ErrSKUNotFound), errors.Is(err, omsDomain.ErrPickingSessionNotFound),
		errors.Is(err, omsDomain.ErrPickScanNotFound), errors.Is(err, omsDomain.ErrPackingSessionNotFound),
		errors.Is(err, omsDomain.ErrPackScanNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, omsDomain.ErrEmptyLineItems), errors.Is(err, invDomain.ErrInsufficientStock),
		errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, omsDomain.ErrLineItemNotOnOrder),
		errors.Is(err, omsDomain.ErrOrderNotStockReducing), errors.Is(err, omsDomain.ErrDuplicateExternalOrder),
		errors.Is(err, omsDomain.ErrPickingSessionNotActive), errors.Is(err, omsDomain.ErrOrderAlreadyFulfilled),
		errors.Is(err, omsDomain.ErrPickScanCodeOrSKURequired), errors.Is(err, omsDomain.ErrInvalidPickScanActionType),
		errors.Is(err, omsDomain.ErrPickLineItemRequired), errors.Is(err, omsDomain.ErrLineItemSKUMismatch),
		errors.Is(err, omsDomain.ErrInvalidLineItemQuantity), errors.Is(err, omsDomain.ErrInvalidPrice),
		errors.Is(err, omsDomain.ErrPriceRequiredForNewItem), errors.Is(err, omsDomain.ErrBulkUpdateOrderIDsRequired),
		errors.Is(err, omsDomain.ErrBulkUpdateNoFieldsGiven), errors.Is(err, omsDomain.ErrOrderChangesHappenInPacking), errors.Is(err, omsDomain.ErrInvalidPackScanActionType),
		errors.Is(err, omsDomain.ErrSubstituteLinesRequired), errors.Is(err, omsDomain.ErrInvalidSubstituteQuantity),
		errors.Is(err, omsDomain.ErrSubstitutionCoverageMismatch), errors.Is(err, omsDomain.ErrInvalidRefundQuantity),
		errors.Is(err, omsDomain.ErrRefundRequiresLinkedSKU), errors.Is(err, pmsDomain.ErrWeightRequiredForPerWeightSKU),
		errors.Is(err, pmsDomain.ErrInvalidCheckDigit), errors.Is(err, omsDomain.ErrPickingSessionAlreadyCompleted),
		errors.Is(err, omsDomain.ErrPackingSessionNotActive), errors.Is(err, omsDomain.ErrPackingSessionAlreadyCompleted),
		errors.Is(err, omsDomain.ErrOrderNotPackable), errors.Is(err, omsDomain.ErrPOSOrderNotFulfillable), errors.Is(err, omsDomain.ErrOrderAlreadyPacked), errors.Is(err, omsDomain.ErrOrderNotPickable),
		errors.Is(err, omsDomain.ErrPickListOrderIDsRequired), errors.Is(err, omsDomain.ErrPickListTooManyOrders),
		errors.Is(err, omsDomain.ErrPackScanCodeOrSKURequired), errors.Is(err, omsDomain.ErrPackScanSKUMismatch),
		errors.Is(err, omsDomain.ErrPackQuantityExceedsOrdered):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, omsDomain.ErrPickingSessionLocked), errors.Is(err, omsDomain.ErrPackingSessionLocked),
		errors.Is(err, omsDomain.ErrOrderPickingInProgress), errors.Is(err, omsDomain.ErrOrderPackingInProgress),
		errors.Is(err, omsDomain.ErrPickingSessionHasPendingChanges):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	case errors.Is(err, omsDomain.ErrPermissionDenied):
		middleware.FailErr(c, http.StatusForbidden, "FORBIDDEN", err)
	default:
		log.Printf("oms: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
