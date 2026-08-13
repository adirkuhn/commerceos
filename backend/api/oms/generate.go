// Package oms holds the OpenAPI spec (source of truth) for sales orders
// (domain/oms) as staff use them from the admin UI -- list/get/create/
// update/delete. The POS's own push path (ReceivePOSSalesOrderUseCase) is
// exposed separately, under api/pos, since the POS authenticates
// differently (its lm2 license, not a human JWT) -- see
// middleware.POSLicense. Run `go generate ./...` after editing openapi.yaml
// to regenerate oms.gen.go -- never hand-edit that file.
package oms

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
