// Package reports holds the OpenAPI spec (source of truth) for the Reports
// module and the generated types/gin ServerInterface derived from it. Run
// `go generate ./...` after editing openapi.yaml to regenerate
// reports.gen.go — never hand-edit that file.
package reports

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
