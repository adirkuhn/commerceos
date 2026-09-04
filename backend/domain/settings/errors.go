package settings

import "errors"

// ErrInvalidTargetMarkupBasisPoints guards
// TenantSettings.DefaultTargetMarkupBasisPoints: must be >= 0. Unlike
// margin, markup (profit / cost) has no upper bound.
var ErrInvalidTargetMarkupBasisPoints = errors.New("target markup basis points must be at least 0")

// ErrInvalidBundleConfidenceThreshold guards
// TenantSettings.BundleConfidenceThreshold: it's a fraction, must be within
// 0-1 inclusive.
var ErrInvalidBundleConfidenceThreshold = errors.New("bundle confidence threshold must be between 0 and 1")
