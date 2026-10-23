// Package quotes holds the OpenAPI spec (source of truth) for saved B2B
// quotes. JWT-protected, mounted under /api/v1. Run `go generate ./...`
// after editing openapi.yaml -- never hand-edit quotes.gen.go.
package quotes

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
