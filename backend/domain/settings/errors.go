package settings

import "errors"

// ErrInvalidTargetMarkupBasisPoints guards
// TenantSettings.DefaultTargetMarkupBasisPoints: must be >= 0. Unlike
// margin, markup (profit / cost) has no upper bound.
var ErrInvalidTargetMarkupBasisPoints = errors.New("target markup basis points must be at least 0")

// ErrStorefrontKeyNotFound is returned by FindTenantIDByStorefrontKeyHash
// when no tenant's stored hash matches.
var ErrStorefrontKeyNotFound = errors.New("storefront key not found")
