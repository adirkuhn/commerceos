// Package collections holds the OpenAPI spec (source of truth) for the
// admin Collections module and the generated types/gin ServerInterface.
// Run `go generate ./...` after editing openapi.yaml -- never hand-edit
// collections.gen.go.
package collections

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
