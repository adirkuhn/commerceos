// Package moneybird holds the OpenAPI spec (source of truth) for the
// Moneybird bookkeeping integration: connecting it, its settings, and
// sending quotes (estimates) and sales orders (external sales invoices) to
// it. JWT-protected, mounted under /api/v1. Run `go generate ./...` after
// editing openapi.yaml to regenerate moneybird.gen.go -- never hand-edit it.
package moneybird

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
