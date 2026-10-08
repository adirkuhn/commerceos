// Package bolconfirmations holds the OpenAPI spec (source of truth) for
// reading whether a Bol order's shipment was confirmed on Bol -- see this
// package's openapi.yaml. JWT-protected, mounted under /api/v1 like every
// other admin module. Run `go generate ./...` after editing openapi.yaml to
// regenerate bolconfirmations.gen.go -- never hand-edit that file.
package bolconfirmations

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
