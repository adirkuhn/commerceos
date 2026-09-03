package apikeys

import "errors"

// ErrNotFound covers both "no key with this ID" (Revoke) and "no active
// key with this hash" (FindActiveTenantIDByHash) -- a revoked key resolves
// the same as a never-existing one, on purpose: a caller must never learn
// which reason it failed for.
var ErrNotFound = errors.New("api key not found")

var ErrNameRequired = errors.New("name is required")
