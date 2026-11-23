// Package assistant holds the OpenAPI spec (source of truth) for the admin
// AI assistant: queued requests, their conversation, replying and
// cancelling. JWT-protected, admin roles only, mounted under /api/v1. Run
// `go generate ./...` after editing openapi.yaml -- never hand-edit
// assistant.gen.go.
package assistant

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
