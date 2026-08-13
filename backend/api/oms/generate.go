// Package oms holds the OpenAPI spec (source of truth) for the read side of
// sales orders (domain/oms) -- staff viewing what came in from POS (or,
// later, any other channel). ReceivePOSSalesOrderUseCase (the only write
// path today) is exposed separately, under api/pos, since the POS
// authenticates differently (its lm2 license, not a human JWT) -- see
// middleware.POSLicense. Run `go generate ./...` after editing openapi.yaml
// to regenerate oms.gen.go -- never hand-edit that file.
package oms

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
