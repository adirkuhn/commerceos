// Package scannerauth holds the OpenAPI spec (source of truth) for the one
// public, unauthenticated endpoint in this codebase: a handheld scanner
// claiming a pairing code. Deliberately its own module mounted outside the
// JWT-protected /api/v1 router group in cmd/api/main.go -- the scanner has
// no token yet, that's the whole point. Run `go generate ./...` after
// editing openapi.yaml to regenerate scannerauth.gen.go — never hand-edit
// that file.
package scannerauth

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
