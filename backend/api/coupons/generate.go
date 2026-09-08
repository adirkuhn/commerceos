// Package coupons holds the OpenAPI spec (source of truth) for the admin
// Coupons API -- CRUD over tenant-wide cart discount codes (see
// domain/coupons). JWT-protected, mounted under /api/v1 like every other
// admin module. Run `go generate ./...` after editing openapi.yaml to
// regenerate coupons.gen.go -- never hand-edit that file.
package coupons

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
