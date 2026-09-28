// Package theme holds the OpenAPI spec (source of truth) for the storefront
// theme system -- admin theme library, the storefront-hosted editor, and the
// public storefront theme reads -- plus the generated types and gin
// ServerInterface. The spec's component props schemas (x-commerceos-component)
// are also the backend's component catalog. Run `go generate ./...` after
// editing openapi.yaml -- never hand-edit theme.gen.go.
package theme

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
