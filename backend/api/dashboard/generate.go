// Package dashboard holds the OpenAPI spec (source of truth) for the admin
// dashboard's summary-figures endpoint (application/dashboard). Run
// `go generate ./...` after editing openapi.yaml to regenerate
// dashboard.gen.go -- never hand-edit that file.
package dashboard

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
