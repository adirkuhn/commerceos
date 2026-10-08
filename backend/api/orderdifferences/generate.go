// Package orderdifferences holds the OpenAPI spec (source of truth) for
// the Stock Issues page's "Order differences": where an imported order
// differs from the channel's own order -- see this package's openapi.yaml.
// JWT-protected, mounted under /api/v1. Run `go generate ./...` after
// editing openapi.yaml to regenerate orderdifferences.gen.go -- never
// hand-edit that file.
package orderdifferences

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
