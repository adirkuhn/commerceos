// Package suppliers holds the OpenAPI spec (source of truth) for the
// Suppliers module and the generated types/gin ServerInterface derived
// from it. Run `go generate ./...` after editing openapi.yaml to
// regenerate suppliers.gen.go — never hand-edit that file.
package suppliers

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
