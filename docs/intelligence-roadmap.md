# CommerceOS Intelligence & Recommendation Roadmap

Source of truth for the multi-phase plan toward forecasting, price
elasticity, promotion analysis, and recommendations. Update the Status
column as slices land — each slice is its own commit.

## Phase 0 — Data Foundation & Business Intelligence

### Data foundation

| Item | Status | Notes |
|---|---|---|
| Sales transaction history | Done (pre-existing) | `domain/oms.SalesOrder` / `SalesOrderLineItem` already immutable, snapshot price/tax at sale time. |
| Product history | **Done (v1)** — `c5a9238`, frontend `47120de` | New `product_history` table (shared by products+variants), `domain/pms.ProductHistoryRepository`. `UpdateProductUseCase`/`UpdateVariantUseCase` record a row when name/status actually change, atomic with the write. `GET /pms/products/{id}/history` + `/pms/variants/{id}/history`. v1 scope only: name+status, not a generic diff of every field. Frontend: read-only card on the product edit page; variant-level history has the API but no UI yet. |
| Price history | **Done** — `269bf37` | `domain/pms.SKUPriceHistory`, effective-dated interval table (`sku_price_history`), written on create/update/sale-expiry. |
| Sale/promotion history | **Done (v1)** — `c2d28ca`, frontend `a90d30f`. Auto-apply: `ea63d85`, frontend `c67baa6` | New `domain/promotions` bounded context: `Promotion` + `PromotionSKU`, CRUD API + frontend page (`/promotions` nav entry). v1 discount types only `percentage` / `fixed_amount`. BOGO, bundle, quantity, category, customer-segment, coupon types **not built**. A promotion now auto-applies while active: a scheduled job (`promotion_activation`, 1min) drives each covered SKU's existing `SalePriceMinorUnits`/`SaleEndDate`, restoring whatever was there before once the promotion ends. Promotion always wins over a pre-existing manual sale price while active -- `CreatePromotionUseCase` warns (doesn't block) when that conflict exists. |
| Cost history | Already covered | `sku_cost_price_history` (daily cron snapshot) + `SetManualCostPriceUseCase`/`ClearCostPriceOverrideUseCase` (source-attributed override). Minor gap: manual override can take up to 24h to appear in history (next cron run) — acceptable for v1, not hardened. |
| Inventory history | Already covered | `domain/inventory.StockMovement` is already a full immutable append-only ledger (SKU, warehouse, batch, delta, reason, occurred_at). Nothing built. |
| Customer activity/events | **Not started** | Scope undefined — needs a decision on what counts as a trackable event (page view, cart add, order, login, etc.) before any design work. |

### Reports

| Report | Status | Notes |
|---|---|---|
| Sales & Revenue Reports | **Partial** — `cc71871` | `GET /reports/sales-summary`: revenue, order count, units sold, AOV for an arbitrary `[from, to)` window, compared against the immediately preceding period of equal length, with % change (nil when previous baseline is 0). Missing from the original wishlist: discounts/refunds/taxes/shipping breakdown, multiple named periods (week/quarter/year presets), YoY. |
| Product Performance | **Done** — `342d13c`, margin added in `fa5049b` | `GET /reports/product-performance`: per-SKU units sold, revenue, distinct order count, first/last sale date, profit, margin, ranked by revenue, for an arbitrary range. Profit/margin nil together when there's no cost basis (adhoc line item, SKU since deleted, or cost price never computed). |
| Product Performance Matrix (sales × margin quadrants) | **Done** — `cdb3647`, frontend `a011215` | `GET /reports/product-performance-matrix`: pure composition over the Product Performance report, no new query. Classifies each SKU with a known margin into winners/stars/hidden_gems/problems, split around the tenant's own median revenue/margin for the period (not a fixed threshold). |
| Slow-Moving Products | **Done** — `8d5ef77`, frontend `f3bb632` | `GET /reports/slow-moving-products`: every SKU with stock on hand and zero units sold in [from, to). Pure composition (no new query) over `ListStockLevelsUseCase` (inventory) + the Product Performance report, capped to the first 500 stocked SKUs (`ListStockLevelsUseCase`'s own page-size ceiling) -- v1 limitation for very large catalogs. |
| Product Lifecycle | **Done** — `d9942d4`, frontend `fd73b8c` | `GET /reports/product-lifecycle`: classifies every SKU sold in the current or immediately preceding period (new/growing/stable/declining/dormant) by comparing quantity sold across the two. Pure composition over Product Performance, called twice -- no new query. |
| Category Analysis | **Done** — `7394a4c`, frontend `0325581` | `GET /reports/category-performance`: revenue/units/order-count per category for line items sold in [from, to), ranked by revenue. LEFT JOINs sku→variant→product→category so unmatched/uncategorized items roll into one "Uncategorized" bucket instead of being dropped. |
| Customer Reports | **Partial** — `8307b52`, frontend `35b13ef` | `GET /reports/customer-performance`: order count, revenue, AOV, first/last order per customer, ranked by revenue. Guest orders (no linked customer) excluded. Frontend: Top Customers panel on Reports. Missing from the original wishlist: new-vs-returning split, purchase frequency, lifetime revenue (needs an all-time lookback query, not just the selected period). |
| Customer Cohorts | **Done** — `611d8a9`, frontend `923726d` | `GET /reports/customer-cohorts`: acquisition cohorts by month of each customer's all-time first order (not just first order in the window), with retention across subsequent months up to `to`. New `domain/oms.CustomerCohortRepository` port. |
| Sales Trends & Seasonality | **Done** — `f7ea714`, frontend `6642945` | `GET /reports/sales-trends`: revenue/orders/units bucketed by day/week/month (`points`), plus revenue/orders summed by weekday across the whole period (`by_weekday`, the seasonality half). New `domain/oms.SalesTrendRepository` port. |
| Promotion Analytics / Opportunities | **Done** — `d415ada`, frontend `c64904e` | `GET /reports/promotions/{promotionId}/performance`: a promotion's covered-SKU revenue/units during its active window (capped at "now" if ongoing, or its actual end date once ended) vs. the immediately preceding period of equal length. New `domain/oms.PromotionPerformanceRepository` port. |
| Inventory + Sales Report | **Done** — `b6f70a9`, frontend `5e39249` | `GET /reports/inventory-sales`: stock on hand joined with units sold in [from, to) for every SKU with stock history. Unlike Slow-Moving Products, unfiltered -- every stocked-or-sold SKU appears, with a turnover rate (units sold / stock on hand, nil when stock is 0). |
| Profitability Reports | **Done** — `00ae483`, frontend `53868aa` | `GET /reports/profitability`: revenue, COGS, profit, margin for [from, to). Profit/margin computed only over `cost_known_revenue_minor_units` (the subset of revenue whose SKU has a known cost price) -- unknown-cost line items are excluded rather than treated as zero cost, so margin never overstates. |
| Business Insights Dashboard (rule-based) | **Done** — `f99a944`, frontend `e32b931` | `GET /reports/business-insights`: new `application/insights` package, pure composition over sales-summary/profitability/product-performance-matrix/slow-moving/inventory-sales. Fixed rule set (v1): revenue decline, slow-moving stock, incomplete cost-price data, hidden-gem/problem products, restock risk (SKU turned over >=100% of stock on hand). Distinct from `application/dashboard` (fixed "this month" snapshot, not a rules engine). |

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

## Frontend status

- **Nav structure**: Promotions and Reports (and all future Intelligence & Recommendation screens) are grouped under an "Intelligence" section header in the sidebar, not flat top-level items — commit `75c177e`.
- **Promotions**: real page at `/promotions` — list/search/delete, create/edit with SKU-line picker (resolves SKU code via `resolveBarcode`, showing product name not a bare ID). Wired to the real backend. Commit `a90d30f`.
- **Reports**: `src/pages/Reports.tsx` rewired to real data — commit `96c9786`. KPI row (revenue/orders/units/AOV with real period-over-period %) from `/reports/sales-summary`; Top Products panel from `/reports/product-performance`; Revenue by Channel reuses the existing `/dashboard/stats` endpoint (labeled "this month" since that endpoint has no date-range param, unlike the period selector). The old Revenue-vs-Cost and Inventory-Turnover charts were removed outright rather than wired to fake data — no backend time-series or stock-turn endpoint exists to back them.
- **SKU price history**: read-only "Price history" card on the product edit page's flat-SKU form (one row per effective-dated interval, open interval marked "Current") — commit `9a1f4af`. Not yet shown for variant-mode SKUs (same scope limit as the pre-existing margin-insight card it sits next to).
- **Product Performance Matrix**: quadrant panel on the Reports page (winners/stars/hidden_gems/problems, period-aware, median shown as caption) — commit `a011215`.
- **Customer Performance**: Top Customers panel on the Reports page — commit `35b13ef`.
- **Category Analysis**: Revenue by Category table on the Reports page (Uncategorized bucket shown, not hidden) — commit `0325581`.
- **Slow-Moving Products**: table on the Reports page, sorted by stock on hand descending — commit `f3bb632`.
- **Profitability**: revenue/COGS/profit/margin card on the Reports page, with a caption noting when cost-known revenue is smaller than total revenue, and an empty state when nothing in the period has a known cost — commit `53868aa`.
- **Inventory + Sales**: table on the Reports page (stock on hand, units sold, turnover %), sorted by turnover descending, "n/a" turnover for a SKU with zero stock on hand — commit `5e39249`.
- **Business Insights**: warning/info banners above the report cards, one per fired rule, section hidden entirely when nothing fires — commit `e32b931`.
- **Sales Trends & Seasonality**: line chart of revenue bucketed by day/week/month (switchable) + bar chart of revenue by weekday — commit `6642945`.
- **Product Lifecycle**: 5-column quadrant-style grid (new/growing/stable/declining/dormant), same visual pattern as the Performance Matrix — commit `fd73b8c`.
- **Customer Cohorts**: cohort-by-month x months-since-acquisition retention table, color-intensity badge per cell — commit `923726d`.
- **Product history**: read-only "Change history" card on the product edit page (name/status changes, most recent first) — commit `47120de`.
- **Promotion auto-apply warning**: creating a promotion whose target SKU already has a manual sale price shows an alert after creation (doesn't block) — commit `c67baa6`.
- **Promotion performance**: "Performance" card on the promotion edit page (revenue/units during vs. before), shown only for an already-saved promotion — commit `c64904e`.

**Phase 0 is now fully complete** except customer activity/events history, which the user explicitly deferred (scope undefined -- needs a decision on what counts as a trackable event before any design work).

Every Phase 0 report on the original roadmap list is now built end-to-end except Promotion Analytics / Opportunities, which needs real promotion usage data first (promotions have no auto-apply/usage tracking yet).

Every Phase 0 backend endpoint built this session now has a real frontend surface. No more backend/frontend gaps to close for what's shipped so far.

## Architecture notes for continuing this work

- Every new domain/report follows existing patterns exactly: `domain/<x>` (entities + repository port) → `application/<x>` (use cases, TDD) → `infrastructure/repositories` (GORM impl + integration test) → `api/<x>/openapi.yaml` (contract, source of truth) → `infrastructure/handlers/<x>` (thin gin handler) → wired in `cmd/api/main.go`.
- New read-only report queries prefer a **new narrow repository interface** (e.g. `domain/oms.SalesReportRepository`, `ProductPerformanceRepository`) over extending an existing one used elsewhere (e.g. `DashboardSalesReportRepository`) — keeps unrelated call sites/tests from having to change.
- All new tables: `tenant_id UUID NOT NULL` + RLS policy + `FORCE ROW LEVEL SECURITY`, per `CLAUDE.md`.
- Every slice: TDD (red → green), unit tests (mocked repos) + integration tests (testcontainers, real Postgres, RLS-isolation proven over the non-superuser app role) + handler tests, full `go build`/`go vet`/`go test` (with and without `-tags integration`) before commit.
