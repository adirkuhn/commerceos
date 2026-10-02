// Package weightportions holds the OpenAPI spec (source of truth) for
// splitting a weight-pool bulk SKU into sellable packages -- see this
// package's openapi.yaml for why it's a separate small module. JWT-
// protected, mounted under /api/v1 like every other admin module. Run
// `go generate ./...` after editing openapi.yaml to regenerate
// weightportions.gen.go -- never hand-edit that file.
package weightportions

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
