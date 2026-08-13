// Package pos holds the OpenAPI spec (source of truth) for the POS desktop
// integration surface: pulling the product catalog and stock levels
// (both with an incremental "changed since" filter), and pushing a
// completed in-store sale. Every route here is authenticated by
// middleware.POSLicense (the caller's lm2 license, verified directly --
// see infrastructure/lm2license), never the human lm2 JWT, so this is
// mounted on its own router group outside /api/v1's normal JWT middleware
// -- see cmd/api/main.go. Run `go generate ./...` after editing
// openapi.yaml to regenerate pos.gen.go -- never hand-edit that file.
package pos

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
