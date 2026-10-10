package reports

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, appreports.ErrToBeforeFrom), errors.Is(err, appreports.ErrInvalidGranularity), errors.Is(err, appreports.ErrSKUIDRequired):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, promotionsDomain.ErrPromotionNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	default:
		log.Printf("reports: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
