# CommerceOS Intelligence & Recommendation Roadmap

Source of truth for the multi-phase plan toward forecasting, price
elasticity, promotion analysis, and recommendations. Update the Status
column as slices land — each slice is its own commit.

## Phase 0 — Data Foundation & Business Intelligence

### Data foundation

| Item | Status | Notes |
|---|---|---|
| Sales transaction history | Done (pre-existing) | `domain/oms.SalesOrder` / `SalesOrderLineItem` already immutable, snapshot price/tax at sale time. |
| Product history | Not started | No change-audit trail on product/variant edits yet. |
| Price history | **Done** — `269bf37` | `domain/pms.SKUPriceHistory`, effective-dated interval table (`sku_price_history`), written on create/update/sale-expiry. |
| Sale/promotion history | **Done (v1)** — `c2d28ca` | New `domain/promotions` bounded context: `Promotion` + `PromotionSKU`, CRUD API. v1 discount types only `percentage` / `fixed_amount`. BOGO, bundle, quantity, category, customer-segment, coupon types **not built**. No auto-apply to SKU price yet — promotions are defined/managed only. |
| Cost history | Already covered | `sku_cost_price_history` (daily cron snapshot) + `SetManualCostPriceUseCase`/`ClearCostPriceOverrideUseCase` (source-attributed override). Minor gap: manual override can take up to 24h to appear in history (next cron run) — acceptable for v1, not hardened. |
| Inventory history | Already covered | `domain/inventory.StockMovement` is already a full immutable append-only ledger (SKU, warehouse, batch, delta, reason, occurred_at). Nothing built. |
| Customer activity/events | **Not started** | Scope undefined — needs a decision on what counts as a trackable event (page view, cart add, order, login, etc.) before any design work. |

### Reports

| Report | Status | Notes |
|---|---|---|
| Sales & Revenue Reports | **Partial** — `cc71871` | `GET /reports/sales-summary`: revenue, order count, units sold, AOV for an arbitrary `[from, to)` window, compared against the immediately preceding period of equal length, with % change (nil when previous baseline is 0). Missing from the original wishlist: discounts/refunds/taxes/shipping breakdown, multiple named periods (week/quarter/year presets), YoY. |
| Product Performance | **Done** — `342d13c`, margin added in `fa5049b` | `GET /reports/product-performance`: per-SKU units sold, revenue, distinct order count, first/last sale date, profit, margin, ranked by revenue, for an arbitrary range. Profit/margin nil together when there's no cost basis (adhoc line item, SKU since deleted, or cost price never computed). |
| Product Performance Matrix (sales × margin quadrants) | **Done** — `cdb3647` | `GET /reports/product-performance-matrix`: pure composition over the Product Performance report, no new query. Classifies each SKU with a known margin into winners/stars/hidden_gems/problems, split around the tenant's own median revenue/margin for the period (not a fixed threshold). |
| Slow-Moving Products | Not started | |
| Product Lifecycle | Not started | |
| Category Analysis | Not started | |
| Customer Reports | Not started | |
| Customer Cohorts | Not started | |
| Sales Trends & Seasonality | Not started | |
| Promotion Analytics / Opportunities | Not started | Needs promotion-performance query (before/during/after comparison) once promotions have real usage data. |
| Inventory + Sales Report | Not started | |
| Profitability Reports | Not started | |
| Business Insights Dashboard (rule-based) | Not started | Distinct from the existing `application/dashboard` module (which is a fixed "this month" snapshot screen, not a rules/insights engine). |

## Phase 1 — Sales Intelligence & Association Rules
Not started. Sales aggregation, RFM, FP-Growth, frequently-bought-together.

## Phase 2 — Demand & Inventory Intelligence
Not started. Demand forecasting, reorder prediction, stockout prediction, product substitution.

## Phase 3 — Customer & Pricing Intelligence
Not started. Customer segmentation, churn prediction, sequential pattern mining, price elasticity.

## Phase 3.5 — Recommendation Measurement
Not started. Recommendation analytics (impressions/CTR/conversion), A/B testing. Must exist before Phase 4 ships, so the engine's output can be judged.

## Phase 4 — CommerceOS Recommendation Engine
Not started. Combines RFM + demand signals + association rules + price elasticity into ranked recommendations; dynamic bundles (margin/inventory/elasticity-aware).

---

## Architecture notes for continuing this work

- Every new domain/report follows existing patterns exactly: `domain/<x>` (entities + repository port) → `application/<x>` (use cases, TDD) → `infrastructure/repositories` (GORM impl + integration test) → `api/<x>/openapi.yaml` (contract, source of truth) → `infrastructure/handlers/<x>` (thin gin handler) → wired in `cmd/api/main.go`.
- New read-only report queries prefer a **new narrow repository interface** (e.g. `domain/oms.SalesReportRepository`, `ProductPerformanceRepository`) over extending an existing one used elsewhere (e.g. `DashboardSalesReportRepository`) — keeps unrelated call sites/tests from having to change.
- All new tables: `tenant_id UUID NOT NULL` + RLS policy + `FORCE ROW LEVEL SECURITY`, per `CLAUDE.md`.
- Every slice: TDD (red → green), unit tests (mocked repos) + integration tests (testcontainers, real Postgres, RLS-isolation proven over the non-superuser app role) + handler tests, full `go build`/`go vet`/`go test` (with and without `-tags integration`) before commit.
