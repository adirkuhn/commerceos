// Package supplierissues holds the OpenAPI spec (source of truth) for a
// purchase order's supplier issues: the missing, damaged and lost items
// to claim from the supplier, their claim status, and emailing the list.
// A separate small module rather than folded into api/procurement's large
// composed handler. JWT-protected, mounted under /api/v1. Run
// `go generate ./...` after editing openapi.yaml -- never hand-edit
// supplierissues.gen.go.
package supplierissues

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
