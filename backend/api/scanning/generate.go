// Package scanning holds the OpenAPI spec (source of truth) for the
// authenticated scanner-pairing endpoints (generate a code, poll its
// status) and the generated types/gin ServerInterface derived from it.
// Deliberately separate from api/iam's Groups module -- that module's
// routes are all mounted behind an admin-only role guard (see
// infrastructure/handlers/iam.RegisterHandler), but any authenticated role
// should be able to pair their own scanner. Run `go generate ./...` after
// editing openapi.yaml to regenerate scanning.gen.go — never hand-edit
// that file.
package scanning

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
