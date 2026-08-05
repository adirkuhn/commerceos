# CommerceOS — Claude Rules

> **READ FIRST.** These rules are mandatory. Before implementing any feature or fix, run the Pre-Implementation Checklist at the bottom. No exceptions.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend language | Go |
| HTTP router | Gin |
| ORM | GORM |
| Database | PostgreSQL 16 |
| DB migrations | goose (single-file Up/Down SQL) — **no GORM AutoMigrate ever** |
| Frontend | React + TypeScript |
| Testing | Go: testify + httptest; Frontend: Vitest + Testing Library; E2E: Playwright |
| Config loader | joho/godotenv (load .env) + plain `os.Getenv` — no struct-tag magic |
| Auth | JWT HS256 from external server **lm2** — validated in `infrastructure/middleware/auth.go` |
| JWT package | golang-jwt/jwt/v5 |
| Message broker | NATS JetStream |
| Event delivery | Transactional Outbox pattern |

---

## Architecture — DDD + Hexagonal

```
domain/          ← entities, value objects, domain events, repository interfaces (ports), event ports
application/     ← use cases (one file per use case), DTOs, service interfaces
infrastructure/  ← adapters: GORM repos, Gin handlers, NATS publishers/consumers, third-party clients
internal/        ← shared plumbing (config, tenant, txctx) — imported by infra, never by domain
```

**Rules:**
- `domain/` must have **zero** imports from `infrastructure/`, `application/`, or `internal/`.
- Repository interfaces in `domain/` accept only `context.Context` — never `*gorm.DB`.
- Transaction is passed via context using `internal/txctx`. Infrastructure repos call `txctx.FromContext(ctx)` to retrieve it. If absent, they return an error (caller violated the transaction rule).
- Use cases receive ports (interfaces), never concrete adapters.
- Gin handlers are thin: parse request → call use case → map to response. No business logic inside handlers.
- GORM models live in `infrastructure/`. Domain entities are separate structs.
- Cross-domain calls go through use cases, never directly to another domain's repository.

---

## SOLID

- **S** — One reason to change per type/file. Split if a struct has 2+ unrelated responsibilities.
- **O** — Extend via new types / strategy pattern. Do not add `if type == X` branches.
- **L** — Subtypes must honour parent contracts. No panic/return-nothing where interface says return error.
- **I** — Define small, focused interfaces at point of use. No "God interface" with 10+ methods.
- **D** — All dependencies injected via constructor. No `new(ConcreteType)` inside business logic.

---

## Database Rules

### No AutoMigrate
Never call `db.AutoMigrate()`. All schema changes live in versioned migration files.

### Multi-tenancy via PostgreSQL Row Level Security (RLS)

Every tenant-scoped table has a `tenant_id UUID NOT NULL` column with an RLS policy:

```sql
USING (current_setting('app.current_tenant', true) = 'bypass_rls'
    OR tenant_id = NULLIF(current_setting('app.current_tenant', true), 'bypass_rls')::uuid)
```

The `NULLIF` matters: casting the literal string `'bypass_rls'` straight to `::uuid` throws (`invalid input syntax for type uuid`) once RLS is actually enforced — Postgres does not guarantee the `OR` short-circuits before the right-hand cast runs. `NULLIF(...)` evaluates to `NULL` instead of the sentinel string when it matches, and `NULL::uuid` is a valid no-op cast, so the comparison is `NULL` (not an error) while the already-true left side still makes the whole expression `true`.

RLS is only enforced at all because of two more things every table needs, added in `020_app_role_and_force_rls.sql`:
- `ALTER TABLE <table> FORCE ROW LEVEL SECURITY;` — without `FORCE`, RLS does not apply to the table's owner.
- The app must connect as `commerceos_app`, a dedicated `NOSUPERUSER NOBYPASSRLS` role — **not** `POSTGRES_USER`/`commerceos`. Postgres superusers and any `BYPASSRLS` role ignore RLS policies unconditionally, `FORCE` or not. `commerceos` stays superuser and is used only for migrations (`GOOSE_DBSTRING`); `commerceos_app` is what `DB_DSN` (backend/relay/scheduler) must point at.

**All DB access (reads and writes) must go through `tenant.WithTenant`:**

```go
err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
    return productRepo.Create(ctx, &product) // tx is inside ctx via txctx
})
```

`tenant.WithTenant` opens a transaction, runs `SET LOCAL app.current_tenant = ?`, embeds the `*gorm.DB` tx into ctx via `txctx.WithTx`, then calls `fn(ctx)`. Infrastructure repos extract the tx with `txctx.FromContext(ctx)`. The tenant ID comes from JWT claims propagated by the auth middleware. `super_admin` role sets `bypass_rls`.

Never query tenant-scoped tables with a bare `db` — always use `tenant.WithTenant`.

### All Writes Under Transaction (mandatory, no exceptions)

Every `INSERT`, `UPDATE`, `DELETE` — even single-row ops — must run inside a transaction. This is a hard rule independent of RLS.

`tenant.WithTenant` satisfies **both** rules simultaneously (it opens a transaction AND sets the tenant). Every write goes through it:

```go
err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
    if err := variantRepo.Create(ctx, &variant); err != nil {
        return err
    }
    // outbox enqueue extracts same tx from ctx — atomic with the write above
    return outboxRepo.Enqueue(ctx, subject, event.EventID, payload, event.OccurredAt)
})
```

**Forbidden patterns:**
```go
db.Create(&x)           // ❌ no transaction, no RLS
db.Save(&x)             // ❌
tx.Create(&x)           // ❌ only if tx came from outside tenant.WithTenant
```

### Repository pattern
Repositories accept `*gorm.DB` (or a `DBTX` interface) so the caller can pass a transaction.

```go
type OrderRepository interface {
    Create(ctx context.Context, tx *gorm.DB, order *domain.Order) error
}
```

---

## Event-Driven Architecture — NATS JetStream + Outbox

### Core Rules
- Domain events are **part of the domain layer**. Define them in `domain/events/`.
- Use cases raise domain events; they never call NATS directly.
- NATS publisher is an infrastructure adapter implementing a `domain.EventPublisher` port.
- Consumers live in `infrastructure/consumers/`. One consumer struct per event type.

### Outbox Pattern (mandatory for all state-changing use cases)
Publishing to NATS must be atomic with the DB write. Use the transactional outbox:

1. Inside the DB transaction, write the event to an `outbox` table (status = `pending`).
2. A background relay process reads `pending` rows and publishes to JetStream.
3. On successful publish, mark row `published`. On permanent failure, mark `failed`.

```go
// Inside a use case — tenant.WithTenant provides the transaction
err := tenant.WithTenant(ctx, db, func(tx *gorm.DB) error {
    if err := orderRepo.Create(ctx, tx, &order); err != nil {
        return err
    }
    return outboxRepo.Enqueue(ctx, tx, subject, event.EventID, payload, event.OccurredAt)
})
```

Never publish to NATS inside a transaction callback. Never publish without an outbox row.

### Event Schema
Every event must have:
```go
type OrderCreated struct {
    EventID   string    `json:"event_id"`   // UUID, for deduplication
    OccurredAt time.Time `json:"occurred_at"`
    // domain payload...
}
```

Subject naming: `{domain}.{entity}.{past-tense-verb}` — e.g., `commerce.order.created`.

### Consumer Rules
- All consumers must be **idempotent** — duplicate delivery must produce no side effects.
- Use `EventID` to deduplicate (check a `processed_events` table before processing).
- Consumers must **ack only after** successful processing. On error: nack + requeue (JetStream handles retry).
- Consumer failures must not crash the process — log, increment metric, nack.
- No consumer calls another consumer directly. Only through new events or use cases.

### Testing Events
- Unit test: use case enqueues correct outbox row (mock outbox repo).
- Integration test: relay publishes outbox row to real NATS (testcontainers).
- Consumer test: given a NATS message, consumer produces correct DB state.

---

## Auth & RBAC

### Roles (from lm2 JWT claim `role`)
| Role | Access |
|---|---|
| `super_admin` | All tenants, all resources. Sets `bypass_rls`. |
| `customer_owner` | Full access to own tenant |
| `customer_admin` | Full access to own tenant (minus tenant management) |
| `customer_user` | Read-only by default; fine-grained rules grant extra access |

### JWT Claims structure (HS256, signed by lm2)
```go
type Claims struct {
    UserID   string `json:"user_id"`
    TenantID string `json:"tenant_id"` // empty for super_admin cross-tenant calls
    Role     string `json:"role"`
    jwt.RegisteredClaims
}
```

### Fine-grained RBAC
Table `user_permissions (user_id, tenant_id, rule TEXT)`. Rule format: `{system}.{resource}.{action}` — e.g. `pms.product.create`, `pms.sku.update`. Check RBAC in use case layer after role check passes.

### Auth middleware flow
1. `middleware.JWT` extracts + validates Bearer token → stores claims in gin context + propagates tenant to `context.Context`.
2. `middleware.RequireRole(...)` guards route groups by role.
3. Use case reads claims from context via `tenant.IDFromContext` / `tenant.IsSuperAdmin`.

---

## Error Handling

- No silent `_ = err` or empty `catch` blocks.
- Wrap errors with context: `fmt.Errorf("createOrder: %w", err)`.
- Define typed domain errors (`ErrOrderNotFound`, `ErrInsufficientStock`) — do not leak DB errors to HTTP layer.
- Gin handlers map domain errors to HTTP status codes in one place (error middleware).
- Frontend: all API errors surface in UI (toast / inline message). No swallowed promise rejections.

---

## API / Contract Design

- REST: `noun-plural/id` paths. `GET /orders`, `POST /orders`, `PATCH /orders/:id`.
- Versioned from day one: `/api/v1/...`.
- Request bodies validated with a DTO struct + binding tags (`binding:"required,email"`).
- Response envelopes consistent: `{ "data": ..., "error": null }` or `{ "data": null, "error": { "code": "...", "message": "..." } }`.
- No business IDs (auto-increment PKs) exposed in URLs. Use UUIDs.
- Pagination on all list endpoints: `limit` + `cursor` or `page`+`per_page`.

---

## Testing Requirements

### Backend (Go)
- Unit tests for every use case (mock repos with interfaces).
- Integration tests for every repository (real PostgreSQL via testcontainers).
- Handler tests with `httptest` for every route (happy path + error cases).
- Coverage gate: **≥ 80%** on `domain/` and `application/`.

### Frontend (React + TypeScript)
- Unit tests for every hook and utility function (Vitest).
- Component tests for every form and complex UI component (Testing Library).
- No `any` type. TypeScript strict mode on.

### E2E (Playwright)
- Cover every critical user flow (checkout, auth, order management).
- Run against a real stack (Docker Compose).

---

## Third-Party Packages & External APIs

**STOP before adding any external package or integrating any third-party API.**

Ask the user:
1. Is there already a package approved for this?
2. Is there a specific vendor package required (e.g., WooCommerce, MyParcel, Revolut, Stripe, Shopify)?

Do not `go get` or `npm install` any new dependency without explicit user approval.

---

## Pre-Implementation Checklist

Run this before writing any code for a new feature or bug fix:

```
[ ] Architecture layer correct? (no domain → infra import)
[ ] All DB writes inside a transaction? (tenant.WithTenant satisfies this)
[ ] All DB access goes through tenant.WithTenant? (RLS + transaction in one)
[ ] New tenant-scoped table has tenant_id + RLS policy?
[ ] No AutoMigrate called?
[ ] New migration file created for schema changes (goose Up/Down)?
[ ] Errors wrapped + typed? No silent catch?
[ ] Dependencies injected (no new ConcreteType inside business logic)?
[ ] Request DTO validated?
[ ] API path follows REST + versioning conventions?
[ ] Route protected by middleware.JWT + middleware.RequireRole?
[ ] Use case unit tests written? (mock all repo interfaces, cover happy path + each error branch)
[ ] Repository integration tests written? (real PostgreSQL via testcontainers)
[ ] Handler tests written? (httptest, happy path + 401/403/422/404 error cases)
[ ] Frontend hook/utility unit tests written? (Vitest)
[ ] Frontend component tests written? (Testing Library, for every form + complex UI)
[ ] E2E tests written for user-facing flow? (Playwright, real Docker Compose stack)
[ ] Coverage ≥ 80% on domain/ and application/?
[ ] Any new external package → asked user first?
[ ] Any new third-party API (Stripe, Shopify, etc.) → asked user first?
[ ] Domain event defined in domain/events/?
[ ] Use case writes outbox row inside same DB transaction (never direct NATS publish)?
[ ] Consumer is idempotent (deduplication via EventID)?
[ ] Event subject follows naming convention: {domain}.{entity}.{verb}?
[ ] Consumer tests planned?
```

If any box is unchecked, resolve it before implementation starts.
