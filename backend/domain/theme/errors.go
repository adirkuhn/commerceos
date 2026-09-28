package theme

import (
	"errors"
	"fmt"
)

var (
	ErrThemeNotFound         = errors.New("theme not found")
	ErrInvalidThemeName      = errors.New("theme name must be 1-120 characters")
	ErrUnknownPreset         = errors.New("unknown theme preset")
	ErrThemeIsLive           = errors.New("the live theme cannot be deleted; publish another theme first")
	ErrStorefrontNotEnabled  = errors.New("this sales channel does not host a themed storefront")
	ErrNoLiveTheme           = errors.New("this storefront has no published theme yet")
	ErrVersionNotFound       = errors.New("theme version not found")
	ErrVersionSealed         = errors.New("a published theme version cannot be changed")
	ErrVersionNotSealed      = errors.New("only a published theme version can be rolled back to")
	ErrVersionNotInTheme     = errors.New("theme version belongs to another theme")
	ErrNoDraft               = errors.New("theme has no draft")
	ErrRevisionConflict      = errors.New("the draft was changed elsewhere; reload and try again")
	ErrDocumentNotFound      = errors.New("theme document not found")
	ErrDocumentExists        = errors.New("a page with this key already exists")
	ErrDocumentNotDeletable  = errors.New("only pages can be deleted")
	ErrInvalidDocumentKind   = errors.New("invalid theme document kind")
	ErrInvalidDocumentKey    = errors.New("invalid theme document key")
	ErrInvalidDocumentTitle  = errors.New("page title must be 1-120 characters")
	ErrInvalidDocument       = errors.New("invalid theme document")
	ErrInvalidStyle          = errors.New("invalid style")
	ErrInvalidSettings       = errors.New("invalid theme settings")
	ErrInvalidCatalog        = errors.New("invalid component catalog")
	ErrInvalidSessionPurpose = errors.New("invalid theme session purpose")
	ErrSessionInvalid        = errors.New("invalid or expired theme session")
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
