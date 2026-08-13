// Package customers holds the OpenAPI spec (source of truth) for Customer
// CRUD (domain/customers). Run `go generate ./...` after editing
// openapi.yaml to regenerate customers.gen.go -- never hand-edit that file.
package customers

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
