// Package iam holds the OpenAPI spec (source of truth) for the IAM
// group/permission routes and the generated types/gin ServerInterface
// derived from it. Run `go generate ./...` after editing openapi.yaml to
// regenerate iam.gen.go — never hand-edit that file.
package iam

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
