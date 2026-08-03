// Package inventory holds the OpenAPI spec (source of truth) for the
// Inventory module and the generated types/gin ServerInterface derived
// from it. Run `go generate ./...` after editing openapi.yaml to
// regenerate inventory.gen.go — never hand-edit that file.
package inventory

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
