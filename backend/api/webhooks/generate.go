// Package webhooks holds the OpenAPI spec (source of truth) for the single
// unauthenticated webhook-ingestion route and the generated types/gin
// ServerInterface derived from it. Kept as its own package, deliberately
// separate from every protected api/* spec -- see openapi.yaml for why.
// Run `go generate ./...` after editing openapi.yaml to regenerate
// webhooks.gen.go -- never hand-edit that file.
package webhooks

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
