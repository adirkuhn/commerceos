package theme

import (
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrThemeNotFound              = errs.New("THEME_NOT_FOUND", "theme not found")
	ErrInvalidThemeName           = errs.New("THEME_NAME_INVALID", "theme name must be 1-120 characters")
	ErrUnknownPreset              = errs.New("THEME_UNKNOWN_PRESET", "unknown theme preset")
	ErrThemeIsLive                = errs.New("THEME_IS_LIVE", "the live theme cannot be deleted; publish another theme first")
	ErrStorefrontNotEnabled       = errs.New("THEMED_STOREFRONT_NOT_ENABLED", "this sales channel does not host a themed storefront")
	ErrNoLiveTheme                = errs.New("THEME_NO_LIVE", "this storefront has no published theme yet")
	ErrVersionNotFound            = errs.New("THEME_VERSION_NOT_FOUND", "theme version not found")
	ErrVersionSealed              = errs.New("THEME_VERSION_SEALED", "a published theme version cannot be changed")
	ErrVersionNotSealed           = errs.New("THEME_VERSION_NOT_SEALED", "only a published theme version can be rolled back to")
	ErrVersionNotInTheme          = errs.New("THEME_VERSION_NOT_IN_THEME", "theme version belongs to another theme")
	ErrNoDraft                    = errs.New("THEME_NO_DRAFT", "theme has no draft")
	ErrRevisionConflict           = errs.New("THEME_REVISION_CONFLICT", "the draft was changed elsewhere; reload and try again")
	ErrDocumentNotFound           = errs.New("THEME_DOCUMENT_NOT_FOUND", "theme document not found")
	ErrDocumentExists             = errs.New("THEME_PAGE_EXISTS", "a page with this key already exists")
	ErrDocumentNotDeletable       = errs.New("THEME_DOCUMENT_NOT_DELETABLE", "only pages can be deleted")
	ErrInvalidDocumentKind        = errs.New("THEME_DOCUMENT_INVALID_KIND", "invalid theme document kind")
	ErrInvalidDocumentKey         = errs.New("THEME_DOCUMENT_INVALID_KEY", "invalid theme document key")
	ErrInvalidDocumentTitle       = errs.New("THEME_PAGE_TITLE_INVALID", "page title must be 1-120 characters")
	ErrInvalidDocumentTranslation = errs.New("THEME_PAGE_TRANSLATION_INVALID", "invalid page text translation")
	ErrInvalidDocument            = errs.New("THEME_DOCUMENT_INVALID", "invalid theme document")
	ErrInvalidStyle               = errs.New("THEME_STYLE_INVALID", "invalid style")
	ErrInvalidSettings            = errs.New("THEME_SETTINGS_INVALID", "invalid theme settings")
	ErrInvalidCatalog             = errs.New("THEME_CATALOG_INVALID", "invalid component catalog")
	ErrInvalidSessionPurpose      = errs.New("THEME_SESSION_PURPOSE_INVALID", "invalid theme session purpose")
	ErrSessionInvalid             = errs.New("THEME_SESSION_INVALID", "invalid or expired theme session")
)

// DocumentError pinpoints which node made a document invalid, so the
// editor can select it. It unwraps to ErrInvalidDocument.
type DocumentError struct {
	NodeID string
	Reason string
}

func (e *DocumentError) Error() string {
	if e.NodeID == "" {
		return fmt.Sprintf("%s: %s", ErrInvalidDocument, e.Reason)
	}
	return fmt.Sprintf("%s: node %q: %s", ErrInvalidDocument, e.NodeID, e.Reason)
}

func (e *DocumentError) Unwrap() error { return ErrInvalidDocument }
