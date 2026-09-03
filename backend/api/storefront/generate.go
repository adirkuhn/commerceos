// Package storefront holds the OpenAPI spec (source of truth) for
// CommerceOS' authenticated public-storefront API -- endpoints a connected
// storefront (e.g. the WooCommerce plugin) calls directly, authenticated
// with a per-connection storefront key rather than a JWT session. Mounted
// outside the JWT-protected /api/v1 router group in cmd/api/main.go, same
// reasoning as api/scannerauth. Run `go generate ./...` after editing
// openapi.yaml to regenerate storefront.gen.go -- never hand-edit that file.
package storefront

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
