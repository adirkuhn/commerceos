// Package notifications holds the OpenAPI spec (source of truth) for the
// Notifications module and the generated types/gin ServerInterface derived
// from it. Run `go generate ./...` after editing openapi.yaml to
// regenerate notifications.gen.go — never hand-edit that file.
package notifications

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
