package settings

import "errors"

// ErrInvalidTargetMarginBasisPoints guards
// TenantSettings.DefaultTargetMarginBasisPoints: must be 0-9999 (a margin
// strictly under 100%) -- a recommended price is undefined/infinite at
// exactly 100% margin and negative above it.
var ErrInvalidTargetMarginBasisPoints = errors.New("target margin basis points must be between 0 and 9999")
