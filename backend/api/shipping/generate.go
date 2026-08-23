// Package shipping holds the OpenAPI spec (source of truth) for the
// MyParcel shipping-label module (domain/shipping) -- connect flow,
// connection settings + per-carrier overrides, and per-SalesOrder shipment
// creation/label retrieval from the pick/pack screen. Own package/spec,
// separate from api/integrations (Bol's connect/product-override routes
// live there since those are genuinely about integration/connection
// management for a sales channel) since this is a new bounded context
// (domain/shipping), not a sales-channel concern. Run `go generate ./...`
// after editing openapi.yaml to regenerate shipping.gen.go -- never
// hand-edit that file.
package shipping

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
