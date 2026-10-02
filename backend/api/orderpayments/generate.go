// Package orderpayments holds the OpenAPI spec (source of truth) for
// backfilling payment info on imported channel orders -- see this
// package's openapi.yaml. JWT-protected, mounted under /api/v1 like every
// other admin module. Run `go generate ./...` after editing openapi.yaml to
// regenerate orderpayments.gen.go -- never hand-edit that file.
package orderpayments

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
