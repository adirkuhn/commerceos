// Package stockexpiry holds the OpenAPI spec (source of truth) for the
// close-to-expiry screen: batches with stock left by close-to-expiry
// window, and marking a batch as seen ("aware") for one window. A separate
// small module rather than folded into api/inventory's large composed
// handler. JWT-protected, mounted under /api/v1. Run `go generate ./...`
// after editing openapi.yaml -- never hand-edit stockexpiry.gen.go.
package stockexpiry

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
