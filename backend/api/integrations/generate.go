// Package integrations holds the OpenAPI spec (source of truth) for the
// protected integrations endpoints and the generated types/gin
// ServerInterface derived from it. Run `go generate ./...` after editing
// openapi.yaml to regenerate integrations.gen.go — never hand-edit that file.
package integrations

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
