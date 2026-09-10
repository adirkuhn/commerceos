# Aurora — Headless Ecommerce Template

A production-quality, reusable React frontend for a full ecommerce platform. Runs entirely
against an in-memory mock backend — no server required — with a clean repository/API seam so a
real backend (a Go API, per the parent project's direction) can be swapped in later without
touching a single component.

This is a **standalone app**, independent of the rest of this repository (`../frontend`,
`../storefront-frontend`). It has its own `package.json`, its own dependency tree, and is not
wired to the Go backend under `../backend`.

## Stack

React 18 · TypeScript (strict) · Vite · Tailwind CSS · shadcn/ui-style primitives (Radix +
class-variance-authority) · React Router 7 · TanStack Query · Zustand · React Hook Form + Zod ·
Vitest + Testing Library · Playwright

## Getting started

```bash
npm install
cp .env.example .env   # optional — sensible defaults work with no .env at all
npm run dev             # http://localhost:3200
```

Demo account for the login/account flows: **morgan@example.com** / **demo-password**.

## Scripts

| Command | Does |
|---|---|
| `npm run dev` | Start the Vite dev server |
| `npm run build` | Type-check, then build a production bundle |
| `npm run preview` | Preview the production build locally |
| `npm test` | Run the Vitest unit/component suite |
| `npm run test:e2e` | Run the Playwright end-to-end suite (starts its own dev server) |

## Architecture

```
src/
  app/            App shell: providers (QueryClient, Theme, i18n), root Layout
  routes/         Route table (React Router, lazy-loaded per page)
  pages/          One file per route — thin, composes features/components
  features/       Domain hooks (TanStack Query) — products, cart, checkout, orders, ...
  api/            Thin functions the features/ hooks call — the intended import boundary
  repositories/   Repository interfaces + Mock*Repository implementations (the real logic)
  components/
    ui/           shadcn/ui-style primitives (Button, Dialog, Tabs, ...)
    ecommerce/    Product/cart/checkout components (ProductCard, VariantSelector, ...)
    layout/       Header, Footer, Breadcrumbs, Container, ...
    sections/     Page-builder section renderers (Hero, ProductGrid, Banner, ...)
    seo/          Hand-rolled <Seo> (title/meta/OG/JSON-LD), no react-helmet
  stores/         Zustand — cart UI, wishlist, compare, auth, checkout, ui prefs
  services/       analytics.ts and authService.ts abstractions
  themes/         ThemeConfig + ThemeProvider (CSS-var runtime switching)
  plugins/        EcommercePlugin registry + two working example plugins
  i18n/           Lightweight dictionary-based i18n (en-US, es-ES)
  config/         Demo catalog (30 products/8 categories) + homepage section config
  types/          Domain types, independent of any transport/API shape
```

### Layered dependency rule

`components/` and `pages/` import from `features/*/hooks.ts` (TanStack Query) or `stores/*`
(Zustand) — never from `api/` or `repositories/` directly. `features/*` import from `api/`.
`api/` imports from `repositories/`. This keeps the data-fetching seam in one place: swapping
`MockProductRepository` for `ApiProductRepository` (see `repositories/index.ts`) is the only
change needed to point the whole app at a real backend.

### State: server vs. client

- **Server state** (products, categories, orders, reviews, search results, promotions) lives in
  TanStack Query, fetched through `features/*/hooks.ts`.
- **Client-only state** (cart drawer open/closed, wishlist product ids, compare list, checkout
  step, theme/locale preference, mobile nav) lives in Zustand (`stores/`), persisted to
  `localStorage` where it makes sense (cart id, wishlist, compare, recently viewed).

Cart *contents* are a middle case: the mock cart "lives" in `MockCartRepository` (as a real
backend's cart would), fetched via TanStack Query keyed by the `cartId` that Zustand's
`cartStore` owns and persists.

### Theme system

Two themes (`Aurora`, `Meridian`) are defined as typed `ThemeConfig` objects in `themes/themes.ts`
— colors (light + dark), typography, radius, component look. `ThemeProvider` writes the active
theme's tokens onto `:root` as CSS custom properties at runtime; Tailwind's `colors.*` all resolve
through those same variables (see `tailwind.config.js`), so switching theme or dark mode never
needs a rebuild.

### Plugin system

`plugins/registry.ts` is a small, flat registry — no microfrontend machinery. A plugin is just an
object matching `EcommercePlugin` (routes, nav items, product tabs, checkout extensions, account
pages, homepage sections, hooks), registered once at startup (`plugins/index.ts`). Two working
examples ship in `plugins/examples/`:

- **loyalty-points** — adds a "Rewards" tab to every product page, computed from the product's
  own price.
- **gift-wrap** — adds a real, interactive checkbox to checkout's shipping step.

### Page-builder

The homepage (`config/homepage.ts`) and CMS pages (`config/demoData.ts`'s `CMS_PAGES`) are arrays
of `PageSection` — `SectionRenderer` renders each one. Adding a new section type is one new union
member + one renderer component registered in the `switch`; unrecognized types fall through to
whatever a plugin registered via `EcommercePlugin.sections`.

## Demo data

`src/config/demoData.ts` seeds 30 real products (not Lorem Ipsum) across 8 categories and 6
brands, with variants, mixed stock states, mixed review coverage, three sample orders, one demo
customer account, four working coupons, and a live flash sale. `src/config/homepage.ts` composes
the homepage from that data via the page-builder sections above.

## What's real vs. UI-only

Per spec: **no real payment processing** anywhere — the payment method selector is a clean,
swappable UI only. Auth, search, and analytics are each behind a small interface
(`services/authService.ts`, `repositories/types.ts`'s `SearchRepository`,
`services/analytics.ts`) with a working mock/console implementation, so a real backend or vendor
SDK is a drop-in replacement, not a rewrite.

## Testing

- **Unit** (`src/**/__tests__/*.test.ts`): `lib/utils`, cart math, every `Mock*Repository`, the
  plugin registry, Zustand stores.
- **Component** (`src/**/__tests__/*.test.tsx`): forms (address, login, register, review) and
  complex UI (product detail + variant selection, search autocomplete, the full multi-step
  checkout).
- **E2E** (`e2e/*.spec.ts`, Playwright): a full guest checkout, and the login/auth-guard flow —
  run against the real dev server and the real mock repositories, no network mocking.

Run `npm test` for unit/component, `npm run test:e2e` for Playwright (installs its own browser on
first run via `npx playwright install chromium` if needed).
