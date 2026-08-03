// Package settings holds the OpenAPI spec (source of truth) for the
// Tenant Settings module and the generated types/gin ServerInterface
// derived from it. Run `go generate ./...` after editing openapi.yaml to
// regenerate settings.gen.go — never hand-edit that file.
package settings

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
