package apikeys

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

// ErrNotFound covers both "no key with this ID" (Revoke) and "no active
// key with this hash" (FindActiveTenantIDByHash) -- a revoked key resolves
// the same as a never-existing one, on purpose: a caller must never learn
// which reason it failed for.
var ErrNotFound = errs.New("API_KEY_NOT_FOUND", "api key not found")

var ErrNameRequired = errs.New("API_KEY_NAME_REQUIRED", "name is required")
