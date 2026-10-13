package settings

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

// ErrInvalidTargetMarkupBasisPoints guards
// TenantSettings.DefaultTargetMarkupBasisPoints: must be >= 0. Unlike
// margin, markup (profit / cost) has no upper bound.
var ErrInvalidTargetMarkupBasisPoints = errs.New("SETTINGS_INVALID_TARGET_MARKUP", "target markup basis points must be at least 0")

// ErrInvalidBundleConfidenceThreshold guards
// TenantSettings.BundleConfidenceThreshold: it's a fraction, must be within
// 0-1 inclusive.
var ErrInvalidBundleConfidenceThreshold = errs.New("SETTINGS_INVALID_BUNDLE_CONFIDENCE", "bundle confidence threshold must be between 0 and 1")

// ErrInvalidRecommendationsMinSupport guards
// TenantSettings.RecommendationsMinSupport: must be >= 0 (0 means "unset,
// use the default" -- see the field's own doc comment).
var ErrInvalidRecommendationsMinSupport = errs.New("SETTINGS_INVALID_MIN_SUPPORT", "recommendations min support must be at least 0")

// ErrInvalidRecommendationsLookbackDays guards
// TenantSettings.RecommendationsLookbackDays: must be >= 0 (0 means
// "unset, use the default" -- see the field's own doc comment).
var ErrInvalidRecommendationsLookbackDays = errs.New("SETTINGS_INVALID_LOOKBACK_DAYS", "recommendations lookback days must be at least 0")

// ErrInvalidLowStockThreshold guards TenantSettings.LowStockThreshold: must
// be >= 0 (0 means never flag low stock).
var ErrInvalidLowStockThreshold = errs.New("SETTINGS_INVALID_LOW_STOCK_THRESHOLD", "low stock threshold must be at least 0")

// ErrInvalidPickingMode guards TenantSettings.PickingMode: scan or list.
var ErrInvalidPickingMode = errs.New("SETTINGS_INVALID_PICKING_MODE", "picking mode must be scan or list")

// ErrInvalidTimezone guards TenantSettings.Timezone: an IANA zone name
// (e.g. "Europe/Amsterdam") or empty.
var ErrInvalidTimezone = errs.New("SETTINGS_INVALID_TIMEZONE", "timezone must be an IANA zone name such as Europe/Amsterdam")

// ErrInvalidDateFormat guards TenantSettings.DateFormat: DD/MM/YYYY,
// MM/DD/YYYY or YYYY-MM-DD.
var ErrInvalidDateFormat = errs.New("SETTINGS_INVALID_DATE_FORMAT", "date format must be DD/MM/YYYY, MM/DD/YYYY or YYYY-MM-DD")

// ErrInvalidExpiryAlertWindow guards ExpiryAlert.Windows: each must be one
// of ExpiryAlertWindowOptions (90, 30, 15 or 7 days).
var ErrInvalidExpiryAlertWindow = errs.New("SETTINGS_INVALID_EXPIRY_WINDOW", "expiry alert windows must be 90, 30, 15 or 7 days")

// ErrExpiryAlertWindowsRequired: an enabled expiry alert needs at least one
// window.
var ErrExpiryAlertWindowsRequired = errs.New("SETTINGS_EXPIRY_WINDOW_REQUIRED", "pick at least one expiry alert window")
