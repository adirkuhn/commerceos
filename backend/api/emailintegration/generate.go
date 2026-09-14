// Package emailintegration holds the OpenAPI spec (source of truth) for
// connecting a tenant's own outbound SMTP relay -- see this package's
// openapi.yaml for why it's a separate small module rather than folded
// into api/integrations. JWT-protected, mounted under /api/v1 like every
// other admin module. Run `go generate ./...` after editing openapi.yaml to
// regenerate emailintegration.gen.go -- never hand-edit that file.
package emailintegration

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
