// Package purchaseplanning holds the OpenAPI spec (source of truth) for
// the Purchase Planner: what to buy, from which supplier, how much and by
// when -- see this package's openapi.yaml. JWT-protected, mounted under
// /api/v1 like every other admin module. Run `go generate ./...` after
// editing openapi.yaml to regenerate purchaseplanning.gen.go -- never
// hand-edit that file.
package purchaseplanning

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
