// Package reviews holds the OpenAPI spec (source of truth) for the admin
// Reviews moderation API -- list every review across all channels and
// delete one (see domain/reviews). JWT-protected, mounted under /api/v1
// like every other admin module. Submitting/listing a product's own
// reviews from the storefront side is a separate, unauthenticated-by-JWT
// contract -- see api/storefront's listStorefrontReviews/submitStorefrontReview.
// Run `go generate ./...` after editing openapi.yaml to regenerate
// reviews.gen.go -- never hand-edit that file.
package reviews

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
