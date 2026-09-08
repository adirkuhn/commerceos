// Package fx holds the OpenAPI spec (source of truth) for the admin
// Exchange Rates API -- CRUD over the tenant's fx conversion rates (see
// domain/fx). JWT-protected, mounted under /api/v1 like every other admin
// module. Run `go generate ./...` after editing openapi.yaml to
// regenerate fx.gen.go -- never hand-edit that file.
package fx

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
