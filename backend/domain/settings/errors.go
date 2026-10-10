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

// ErrInvalidRecommendationsMinSupport guards
// TenantSettings.RecommendationsMinSupport: must be >= 0 (0 means "unset,
// use the default" -- see the field's own doc comment).
var ErrInvalidRecommendationsMinSupport = errors.New("recommendations min support must be at least 0")

// ErrInvalidRecommendationsLookbackDays guards
// TenantSettings.RecommendationsLookbackDays: must be >= 0 (0 means
// "unset, use the default" -- see the field's own doc comment).
var ErrInvalidRecommendationsLookbackDays = errors.New("recommendations lookback days must be at least 0")

// ErrInvalidPickingMode guards TenantSettings.PickingMode: scan or list.
var ErrInvalidPickingMode = errors.New("picking mode must be scan or list")

// ErrInvalidTimezone guards TenantSettings.Timezone: an IANA zone name
// (e.g. "Europe/Amsterdam") or empty.
var ErrInvalidTimezone = errors.New("timezone must be an IANA zone name such as Europe/Amsterdam")

// ErrInvalidDateFormat guards TenantSettings.DateFormat: DD/MM/YYYY,
// MM/DD/YYYY or YYYY-MM-DD.
var ErrInvalidDateFormat = errors.New("date format must be DD/MM/YYYY, MM/DD/YYYY or YYYY-MM-DD")

// ErrInvalidExpiryAlertWindow guards ExpiryAlert.Windows: each must be one
// of ExpiryAlertWindowOptions (90, 30, 15 or 7 days).
var ErrInvalidExpiryAlertWindow = errors.New("expiry alert windows must be 90, 30, 15 or 7 days")

// ErrExpiryAlertWindowsRequired: an enabled expiry alert needs at least one
// window.
var ErrExpiryAlertWindowsRequired = errors.New("pick at least one expiry alert window")
