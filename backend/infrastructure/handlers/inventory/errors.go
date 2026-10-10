package inventory

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, invDomain.ErrWarehouseNotFound), errors.Is(err, invDomain.ErrBatchNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, invDomain.ErrInvalidMovementType),
		errors.Is(err, invDomain.ErrInsufficientStock), errors.Is(err, invDomain.ErrSKUIsNotACase),
		errors.Is(err, invDomain.ErrSKUIsNotPortioned), errors.Is(err, invDomain.ErrUnsupportedPhotoType),
		errors.Is(err, invDomain.ErrBatchDoesNotBelongToSKU), errors.Is(err, invDomain.ErrBatchNotInWarehouse):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, pmsDomain.ErrSKUNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, invDomain.ErrBatchHasMovements), errors.Is(err, invDomain.ErrDuplicateWarehouseName):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	default:
		log.Printf("inventory: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
