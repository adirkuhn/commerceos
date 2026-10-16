# CommerceOS — Project Overview (AI handoff)

Snapshot: 2026-09-27, ~890 commits, 185 goose migrations, ~1,540 Go files, ~970 TS/TSX files.
Mandatory coding rules live in `CLAUDE.md` (root) — read that before changing anything. This file describes *what exists*.

## 1. What it is

CommerceOS is a **multi-tenant commerce back-office for small/medium merchants** (Netherlands-focused: MyParcel, Bol.com, PDOK, EUR default). One system holds the catalog, stock, purchasing and orders, and syncs them out to sales channels (WooCommerce, Bol.com, own storefront, WhatsApp, a desktop POS). It includes warehouse operations (a paired mobile scanner for receiving/picking/packing), shipping labels, payments, customer messaging, and BI reports.

Identity comes from an external auth server **lm2** (HS256 JWT). Tenancy is enforced by PostgreSQL Row Level Security.

## 2. Repository layout

```
backend/               Go (Gin, GORM, goose, NATS JetStream)
  domain/<ctx>/        entities, value objects, repo interfaces (ports), errors — zero outward imports
  domain/events/       domain event structs
  application/<ctx>/   one use case per file, DTOs
  infrastructure/      adapters: handlers/<ctx> (Gin), repositories (GORM), consumers, publishers,
                       woocommerce, bol, myparcel, stripe, revolut, evolutionapi, pdok, ai, storage, pdf,
                       middleware, db/migrations (goose SQL)
  api/<ctx>/openapi.yaml   contract per bounded context; requests validated against it at runtime
  internal/            config, tenant (WithTenant/RLS), txctx, crypto, idgen, db
  cmd/                 processes (see §4)
frontend/              admin SPA + mobile scanner app (React 18, TS, Vite, Tailwind, React Router 7)
storefront/            (being rebuilt) SSR shop + GrapesJS theme editor — see docs/storefront-architecture.md
docs/intelligence-roadmap.md   BI/forecasting roadmap with per-item status
docker-compose.yml     local stack (Postgres + MinIO are external networks, not defined here)
```

## 3. Core architecture patterns

- **DDD + Hexagonal.** Handlers are thin (parse → use case → envelope). Use cases take ports, never adapters. Cross-context calls go through use cases or events, never another context's repo.
- **Tenancy via RLS.** Every tenant table has `tenant_id` + RLS policy + `FORCE ROW LEVEL SECURITY`. App connects as `commerceos_app` (NOBYPASSRLS); `commerceos` superuser only for migrations. All DB access goes through `tenant.WithTenant(ctx, db, fn)` which opens a tx, `set_config('app.current_tenant')`, and stores tx in ctx (`txctx`). Repos fail if no tx in ctx. `tenant.WithSuperAdmin` sets `bypass_rls` for cross-tenant pollers.
- **Transactional outbox.** State-changing use cases enqueue an outbox row in the same tx. `cmd/relay` publishes pending rows to JetStream (stream `COMMERCEOS_EVENTS`). Subjects: `{domain}.{entity}.{verb}`.
- **Consumers** (hosted in `cmd/scheduler`): idempotent via `processed_events` (EventID), ack after success, NAK with backoff (5s/30s/2m/10m/30m), dead-letter to `dlq.<subject>` after 5 deliveries. Malformed payloads acked+dropped.
- **Response envelope:** `{data, error:{code,message}}`. Paths `/api/v1/...`, UUIDs only.
- **Money:** integer minor units; percentages in basis points. No money library.
- **Secrets:** integration credentials encrypted at rest (`internal/crypto`, `INTEGRATIONS_ENCRYPTION_SECRET`).
- **Auth/RBAC:** roles `super_admin`, `customer_owner`, `customer_admin`, `customer_user` (read-only by default). `middleware.JWT`, `RequireRole`, `RequireRoleForWrites`, `RequirePermission(rule)` (IAM group rules `{system}.{resource}.{action}`).

## 4. Processes (`backend/cmd`)

| Process | Role |
|---|---|
| `api` | HTTP server (all REST + storefront + webhook ingress + POS) |
| `relay` | Outbox → JetStream publisher (~2s poll; 10 retries then `failed`; NATS outages don't count) |
| `scheduler` | Channel sync-schedule poller, cron jobs, and host for all event consumers |
| `webhook-processor` | Cold path for inbound webhooks (queue group, scalable) |
| `webhook-forwarder` | Delivers tenant-configured webhook forwards |
| `backfill-*`, `repair-stock-review-flags`, `repair-partially-received-pos` | One-off maintenance tools |

Cron jobs (`domain/scheduler`): `expired_sale_prices`, `expired_batches`, `cost_price_recalculation`, `resync_all_products_to_channels`, `promotion_activation`, `reconcile_fulfilled_orders`. Admin can enable/disable, set interval, run now.

## 5. Feature inventory by bounded context

### PMS — catalog (`pms`)
- Product (draft/active/archived) → Variant (+attributes) → SKU. Categories, media (local or S3/MinIO), barcode aliases, scale-code aliases (weight-embedded barcodes decoded by `resolve_barcode`).
- SKU unit types: unit / box / kilo. **Kilo SKU = weight pool**, stock counted in grams; sellable portions link via `PortionedFromSKUID`; bulk "type a list of weights → N portion SKUs".
- **Case packs:** `CaseOfSKUID` + `UnitsPerCase`, one level only; split-case moves stock case→units.
- Pricing: fixed or per-weight, sale price + end date, per-currency price overrides, price history (effective-dated).
- **Cost:** moving-average from receipts (landed cost), manual override, cost ledger, cost history, margin insight / margin report; cost frozen on sale lines.
- Product history (name/status changes), duplicate product.
- **Merge** duplicate → existing product (admins only, irreversible, one tx): maps each SKU, moves stock/batches/cost/orders/aliases, deletes duplicate. Products with history can't be deleted (409 → archive).
- Events: `pms.product.changed`, `pms.product.deleted`, `pms.category.changed`, ProductMerged.

### Inventory / WMS (`inventory`)
- Warehouses, Batches (lot + expiry, active/expired), immutable StockMovement ledger (receipt, sale, adjustment, transfer_out/in, damaged, lost, split_out/in), discards.
- Receive, recount, transfer, mark damaged/lost, split case, portion from bulk, discard batch.
- Sale allocation: **FEFO** → FIFO → general ledger (nil batch); expired batches excluded.
- Expiry workflow: expiring-soon list, expired-review queue (write off / extend / dismiss), auto-expire by threshold days.
- Optional "inventory sync rules" tenant setting: batch-expiry-aware sellable stock; source batch's lot/expiry pushed to Woo meta.
- Events: `inventory.stock.changed`, GoodsReceived.

### Procurement + Suppliers (`procurement`, `suppliers`)
- Suppliers CRUD. Purchase orders (draft/sent/partially_received/received/cancelled/needs_review) with line items, units-per-purchased-item, warehouse assigned lazily, currency defaulting from tenant settings.
- **Document parsing:** upload PDF/CSV/XLSX → async job → deterministic parse (CSV/XLSX) or AI extraction (PDF text via gxpdf) → matching pipeline (exact SKU → learned supplier alias → needs review). Create PO from document; create product/supplier inline from a line.
- **Receiving:** invoice upload, reconciliation vs PO, discrepancy queue (quantity mismatch, missing, unexpected, expiry mismatch, damaged), landed cost.
- **Scanner booking:** handheld receiving session (start/resume/pause/complete). Scans are *staged* and applied only on completion (idempotent replay), with receive/correction/damaged/lost actions.

### OMS — sales orders & fulfillment (`oms`)
- SalesOrder sources: pos, manual, woocommerce, bol, whatsapp, internal. Statuses mirror Woo: pending, processing, on-hold, shipped, completed, cancelled, refunded, failed, partially_refunded. on-hold/processing/shipped/completed decrement stock; cancelled/failed restock.
- Activity log, bulk status updates, printable invoices / pick lists.
- **Review queues:** `NeedsStockReview` (Stock Issues page: retry / force-accept), `NeedsSKUReview` (Unmatched SKUs page: link line to SKU), `NeedsManualRefundReview`.
- **Picking (optional)** — scan session or list mode; pick list across orders.
- **Packing (main step)** — scan-to-pack, substitute, freebie, add item, refund line (via Stripe/Revolut refund → `partially_refunded`), then shipping label.
- Session locks (2-min TTL heartbeat, admin take-over), shown in queues.
- Events: `oms.sales_order.status_changed`, `payments.refund.issued`, `payments.refund.manual_review_required`.

### Shipping (`shipping`) — MyParcel
- Connect MyParcel, per-carrier settings/overrides (PostNL, bpost, DPD, DHL variants, UPS). Create/cancel shipment, label PDF/ZPL (browser print), track & trace, export as MyParcel order, reconciliation job, webhooks for status changes.
- Label flow: address confirm (PDOK NL address lookup) → preview → create.
- Delivery price estimate (also used by storefront checkout shipping options).
- "processed" status → non-Woo order `shipped`, Woo order `on-hold`. Event: `shipping.shipment.fulfillment_signal`.

### Sales channels & integrations (`channels`, `integrations`)
- **Connection** per provider: woocommerce, bol, myparcel, whatsapp (Evolution API), stripe, revolut, email (per-tenant SMTP). Separate AI connection (openai/anthropic/gemini/ollama, one active per tenant).
- **SalesChannel** (woocommerce | bol | internal | whatsapp): enrollment mode all/category/tag/manual + per-product override, price adjustment (bp), B2B quantity breaks, fixed currency (FX hybrid: tenant exchange rates + per-SKU currency overrides), default warehouse, payment connection.
- Listings per product/channel, **sync log** (push/import, field diffs, retry/dismiss), sync schedules per entity+direction (products, stock, orders, reconciliation, bol_*).
- **WooCommerce:** OAuth-style connect or manual; product push in 3 shapes (single / separate products per variant / variable+variations), images, categories (connection-scoped links), tax class links, stock + lot/expiry meta, order import (incremental, reconciliation), status + notes pushed back, provider webhook management. Companion WP plugin lives in a separate repo.
- **Bol.com:** Retailer API (OAuth client-credentials), offers, product content submit, commission/profit calc, economic operators, order import, stock sync. Not yet verified against real credentials.
- **Stripe / Revolut:** payment links, refunds; Stripe PaymentIntents for embedded checkout. Payment webhooks mark orders paid (idempotent).
- Event: `integrations.connection.connected`.

### Webhooks (`webhooks`)
- Ingress `POST /webhooks/v1/{tenant_id}/{provider}[/{token}]`: verify signature → publish raw envelope to stream `COMMERCEOS_WEBHOOKS` (`webhooks.received.{provider}`) → DB fallback on publish failure.
- Processor dedupes, persists, dispatches (MyParcel status, Stripe/Revolut payments, WhatsApp inbound), and enqueues **forward rules** (provider+topic → URL); forwarder delivers. Real verifiers: woo, myparcel, whatsapp, stripe, revolut. Stubs (501): bol, moneybird, shopify, lightspeed, sendcloud.

### Internal storefront (`storefront`, `theme`, `reviews`)
- Headless API `/storefront/v1` authed by a channel-scoped publishable **API key**. Products (enrollment-scoped, paginated, category facets), categories, recommendations (FP-Growth), reviews (verified-buyer flag), persisted token carts, coupons, shipping options, Stripe PaymentIntent or hosted payment link, checkout (server re-prices), order lookup, wishlists, passwordless magic-link customer accounts.
- **Theme system:** being rebuilt from scratch (versioned themes, GrapesJS editor, SSR runtime) — see `docs/storefront-architecture.md`. The old Puck/freeform builder and its `theme_pages` tables were removed 2026-09-28.
- Event: `storefront.order.placed`.

### WhatsApp (`whatsapp`)
- Via self-hosted Evolution API (no official API). Inbound commands STOCK/PRICE/EXPIRY/INFO/HELP; stateful chat ordering (ORDER/CART/CHECKOUT/CANCEL, YES/NO) creating customer + sales order + optional payment link.
- Optional AI intent parsing for free text — classifies only read-only intents; replies always come from fixed formatters. Test mode, message log, conversations UI.
- Outbound order-status and delivery notifications (needs customer `PhoneE164`).

### Message Center (`messagecenter`, `messaging`, `notifications`)
- Templates + automation rules (trigger subject + conditions → template) over 10 known event subjects; engine consumer renders and sends via tenant WhatsApp or Email connection (falls back to platform `SMTP_*`). SMS deliberately not built.
- In-app notifications (bell) + per-category email preferences.

### Marketing
- **Promotions:** percentage/fixed, SKU lines, scheduled; activation job writes SKU sale price and restores previous value when ended. **Clearance promotions** tied to a batch end when batch is empty/expired/deleted.
- **Coupons:** percent or fixed with per-currency amounts; applied in storefront cart.
- **Reviews:** moderation list/delete.

### Customers, Tax, FX, Settings, IAM
- Customers + addresses, WhatsApp message history per customer.
- Tax classes (default class) + rates (bp) + external links to Woo tax classes.
- FX exchange rates per currency.
- Settings: tenant (timezone, date format, picking mode, default currency, inventory sync rules), company profile, notification prefs, API keys.
- IAM: groups, members, permission rules, effective permissions, user activity. Users themselves live in lm2 (invites via lm2).
- **Scanner pairing:** desktop mints 6-digit / QR / barcode code → handheld claims it → gets token with same claims (12h, refreshed).

### POS
- `/pos/products|stock-levels|warehouses`, `POST /pos/sales-orders`, authed by RS256 lm2 **license** (CompanyID = tenant). Mounted only if `LM2_LICENSE_PUBLIC_KEY_URL` set. POS app is a separate Windows (Wails) repo `akantio_pos`.

### Reports, insights, dashboard, search
- 19 `/reports/*` endpoints: sales summary/trends, revenue by channel, product performance + matrix (winners/stars/hidden gems/problems), lifecycle, category, slow-moving, stockout risk, inventory+sales, profitability (+trend), promotion performance, customer performance / RFM / cohorts, frequently bought together + recommendations (FP-Growth).
- Business insights rule engine over reports; dashboard monthly stats.
- Global Cmd+K search (products, suppliers, customers, orders; fuzzy, accent-insensitive).
- Roadmap for forecasting / recommendation engine: `docs/intelligence-roadmap.md`.

### AI usage (`infrastructure/ai`)
Pluggable per tenant (OpenAI, Anthropic, Gemini, Ollama) for two jobs only: procurement document line-item extraction, and WhatsApp intent classification. Everything works without AI (deterministic paths).

## 6. Frontends

### Admin (`frontend/`)
- Nav groups (`components/layout/navConfig.ts`): Dashboard · Sell (Sales Orders, Customers, Sales Channels) · Fulfill (Picking, Packing, Stock Issues) · Catalog (Products, Inventory, Warehouses, Unmatched SKUs) · Buy (Suppliers, Procurement) · Marketing (Promotions, Coupons, Reviews, Message Center) · Reports · Connect (Integrations, Webhooks, Sync History) · Admin (Users & Roles, Scheduled Jobs, Settings).
- Routing: `src/routes.ts` + `src/routing` hooks; **every view/record has its own URL** (refresh/Back must work).
- API: per-domain `api.ts` with `fetch` + envelope unwrap; hand-written types; `src/contracts` test checks TS enum unions against backend OpenAPI enums.
- Design system: semantic Tailwind tokens (canvas/surface/ink/accent "postmark ultramarine"), `text-2xs` floor, Dialog/useConfirm, dark mode. Shortcuts: Cmd+K palette, `g`+letter navigation.
- **Scanner app** at `/scanner/*`: full-screen dark kiosk UI for Receive PO, Inventory lookup/actions, Price lookup, Discard, Pick, Pack; camera (zxing) or hardware scanner; separate paired token.
- Tests: Vitest + Testing Library (~210 files); Playwright e2e against real docker stack (needs `E2E_JWT_SECRET`, `E2E_TENANT_ID`).

### Storefront (`storefront/`)
Being rebuilt (2026-09-28): SSR runtime + GrapesJS editor. Architecture and phase plan in `docs/storefront-architecture.md`. Internal channels stay usable headless (REST only) meanwhile.

## 7. Dev environment notes

- Local docker stack is **not** production (prod is a separate remote host). User runs `docker compose build/up` themselves.
- Go verification: run `go vet`/`go test` **with `-tags integration`** too; integration tests use testcontainers Postgres.
- Node: non-interactive shell has Node 18; use nvm Node 24 for Vitest. Playwright pinned to 1.48.x.
- Migrations: goose single-file Up/Down SQL in `backend/infrastructure/db/migrations`; never AutoMigrate. Coordinate migration numbers if parallel sessions are active.

## 8. Working agreement (from the owner)

1. Bug/feature → find root cause first, present problem + proposed fix + why it works, **wait for approval**.
2. Then TDD, clean architecture, clean code.
3. Every API change: OpenAPI contract, validated, documented.
4. No new third-party package or API without asking.
5. Cross-concern state changes raise domain events (no direct use-case-to-use-case calls).
6. Small commits per step, pushed.
7. Files: soft target ~400 lines, split at ~600–800.

## 9. Known gaps / not built

- Bol + MyParcel never tested against real accounts.
- Returns/RMA workflow, full order-lifecycle automation.
- Smart multi-warehouse routing (currently a static default warehouse per channel).
- Webhook verifiers for bol/moneybird/shopify/lightspeed/sendcloud are stubs.
- SMS channel in Message Center.
- Raw USB/network label printing (browser print only).
- Forecasting / recommendation engine phases (see roadmap).
- `frontend/src/pages/Orders.tsx` and `/shipping` are prototype/demo pages, not in the nav.
