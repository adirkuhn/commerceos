// Package theme holds the OpenAPI spec (source of truth) for the admin
// Theme Builder API -- fetching and saving one internal sales channel's
// canvas page documents. JWT-protected, mounted under /api/v1 like every
// other admin module. Run `go generate ./...` after editing openapi.yaml
// to regenerate theme.gen.go -- never hand-edit that file.
package theme

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
