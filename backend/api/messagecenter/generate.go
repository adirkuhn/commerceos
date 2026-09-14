// Package messagecenter holds the OpenAPI spec (source of truth) for the
// Message Center API -- CRUD over merchant-authored message templates (see
// domain/messagecenter). JWT-protected, mounted under /api/v1 like every
// other admin module. Run `go generate ./...` after editing openapi.yaml to
// regenerate messagecenter.gen.go -- never hand-edit that file.
package messagecenter

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
