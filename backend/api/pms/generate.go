// Package pms holds the OpenAPI spec (source of truth) for the PMS module
// and the generated types/gin ServerInterface derived from it. Run
// `go generate ./...` after editing openapi.yaml to regenerate pms.gen.go —
// never hand-edit that file.
package pms

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
