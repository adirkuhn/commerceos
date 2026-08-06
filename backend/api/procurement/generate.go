// Package procurement holds the OpenAPI spec (source of truth) for the
// Procurement module and the generated types/gin ServerInterface derived
// from it. Run `go generate ./...` after editing openapi.yaml to
// regenerate procurement.gen.go — never hand-edit that file.
package procurement

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
