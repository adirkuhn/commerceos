// Package deadletters holds the OpenAPI spec (source of truth) for the
// super_admin dead-letter admin (COM-14): list, read, retry and discard
// events a consumer gave up on. Every route is super_admin only (see
// cmd/api's mounting). Run `go generate ./...` after editing openapi.yaml
// to regenerate deadletters.gen.go -- never hand-edit that file.
package deadletters

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
