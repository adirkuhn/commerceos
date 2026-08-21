// Package scheduler holds the OpenAPI spec (source of truth) for the
// super_admin "Scheduled Jobs" admin page -- global cron jobs
// (domain/scheduler.Job) and every tenant's Woo sync schedules
// (domain/channels.SyncSchedule) in one cross-tenant list, with a run-now
// action for each. Every route here is gated to super_admin only (see
// infrastructure/handlers/scheduler's Register) -- these are
// platform-operational views/actions, not tenant self-service settings.
// Run `go generate ./...` after editing openapi.yaml to regenerate
// scheduler.gen.go -- never hand-edit that file.
package scheduler

//go:generate go tool oapi-codegen -config oapi-codegen-config.yaml openapi.yaml
