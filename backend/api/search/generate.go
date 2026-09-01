// Package search holds the OpenAPI spec (source of truth) for the global
// search (Cmd+K) endpoint (application/search). Run `go generate ./...`
// after editing openapi.yaml to regenerate search.gen.go -- never hand-edit
// that file.
package search

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
