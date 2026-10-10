package theme

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

var (
	notFound = []error{
		theme.ErrThemeNotFound, theme.ErrVersionNotFound, theme.ErrDocumentNotFound, theme.ErrNoLiveTheme,
		channelsDomain.ErrSalesChannelNotFound, theme.ErrAssetNotFound,
	}
	conflict = []error{
		theme.ErrRevisionConflict, theme.ErrThemeIsLive, theme.ErrVersionSealed, theme.ErrDocumentExists,
		theme.ErrNoDraft, theme.ErrStorefrontNotEnabled,
	}
	invalid = []error{
		theme.ErrInvalidThemeName, theme.ErrUnknownPreset, theme.ErrInvalidDocumentKind, theme.ErrInvalidDocumentKey,
		theme.ErrInvalidDocumentTitle, theme.ErrInvalidDocumentTranslation, theme.ErrInvalidDocument, theme.ErrInvalidStyle, theme.ErrInvalidSettings,
		theme.ErrInvalidSessionPurpose, theme.ErrVersionNotSealed, theme.ErrVersionNotInTheme, theme.ErrDocumentNotDeletable,
		theme.ErrUnsupportedAssetType, theme.ErrAssetTooLarge, theme.ErrInvalidAssetAlt, locale.ErrUnsupported,
	}
)

func isAny(err error, targets []error) bool {
	for _, t := range targets {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}

// fail maps a use-case error to its HTTP response; it returns false when
// err is nil so callers can write `if !fail(c, err) {...}`.
func fail(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var docErr *theme.DocumentError
	switch {
	case errors.As(err, &docErr):
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"data": nil, "error": gin.H{
			"code": "VALIDATION_ERROR", "message": docErr.Error(), "node_id": docErr.NodeID,
		}})
	case errors.Is(err, theme.ErrSessionInvalid):
		middleware.FailErr(c, http.StatusUnauthorized, "UNAUTHORIZED", err)
	case isAny(err, notFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case isAny(err, conflict):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	case isAny(err, invalid):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("theme: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return true
}

// failPublic is fail for shopper-facing routes, where "this channel has no
// themed storefront" is simply "not found".
func failPublic(c *gin.Context, err error) bool {
	if errors.Is(err, theme.ErrStorefrontNotEnabled) {
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
		return true
	}
	return fail(c, err)
}
