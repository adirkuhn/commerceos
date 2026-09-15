// Package whatsapp holds the OpenAPI spec (source of truth) for the admin
// WhatsApp conversations API -- the "WhatsApp Web"-style list of every
// conversation (see domain/whatsapp.Message) and one conversation's full
// thread. JWT-protected, mounted under /api/v1 like every other admin
// module. Run `go generate ./...` after editing openapi.yaml to
// regenerate whatsapp.gen.go -- never hand-edit that file.
package whatsapp

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
