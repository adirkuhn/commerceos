// Package integrationscallback holds the OpenAPI spec (source of truth) for
// the single unauthenticated WooCommerce callback route and the generated
// types/gin ServerInterface derived from it. Kept as its own package,
// deliberately separate from api/integrations (the protected ops) — see
// openapi.yaml for why. Run `go generate ./...` after editing openapi.yaml
// to regenerate integrationscallback.gen.go — never hand-edit that file.
package integrationscallback

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
