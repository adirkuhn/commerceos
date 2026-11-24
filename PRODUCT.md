# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

A small team running one specialty food webshop (the owner's own shop; CommerceOS is an internal tool, not sold to other merchants).

- **Owner / admins at a desk:** orders, customers, quotes and price list, catalog, buying from suppliers, purchase planning, promotions, reports, settings and integrations. Long sessions, many tabs, keyboard-heavy.
- **Warehouse staff:** picking and packing queues, stock issues, receiving purchase orders, expiring batches. On warehouse screens, often standing, mid-task.
- **Phone scanner:** a separate dark-only scanner app for scanning at the shelf (out of scope for admin restyles unless named).

## Product Purpose

One back office for a multi-channel shop: orders from WooCommerce, Bol, the own storefront, POS and WhatsApp land in one place; stock is tracked per lot with expiry dates; buying, fulfilment, shipping labels (PostNL, MyParcel), invoicing (Moneybird) and customer messaging run from it. Success means the daily work (pick, pack, ship, restock, sell off expiring stock) gets done fast and without mistakes.

## Positioning

Built for this shop's actual mechanics: lot/expiry-aware stock with clearance promotions, weight-split packages, case-to-unit costing, multi-channel stock and price sync (Woo, Bol with automatic fee-aware pricing), and a scan-first warehouse flow. Generic commerce admins do not model these.

## Operating Context

- Desktop browser in the office; warehouse screens during picking/packing; phones for the scanner.
- Others can see the screen: money amounts are hidden by default and revealed on click (never persisted).
- Three UI languages: English, Brazilian Portuguese, Dutch. Light and dark appearance (System default).
- Shop owner can set per-shop appearance (font, accent, sidebar colour, corners, text size, density).

## Capabilities and Constraints

- React + TypeScript + Tailwind admin in `frontend/`; storefront is a separate app in `storefront/`.
- Every page has its own URL; refresh and Back must work.
- Amount display goes through `<Money>` (hidden by default).
- Decision 2026-11-24: the admin moves to one new design. The classic and depot designs and the switch between them are retired.

## Brand Commitments

- Name: CommerceOS. Mark: the stock-tag-with-barcode BrandMark (chosen by Claude 2026-09-23; not a user-made brand, may evolve).
- Readability matters: the user rejected Barlow as hard to read and picked Atkinson Hyperlegible Next for depot.

## Evidence on Hand

- Real local data is thin (test orders and products); screenshots of earlier designs in `frontend/e2e-screenshots/`.
- No testimonials, customers or marketing claims exist; none to be invented.

## Product Principles

1. Daily work first: what needs doing today is one glance from anywhere.
2. Fewer mistakes over fewer clicks: scan and confirmation flows guard stock and shipping.
3. Private by default: revenue and amounts stay hidden from onlookers.
4. One place, many channels: the channel an order came from is always visible but never forks the workflow.

## Accessibility & Inclusion

- Legible type over stylish type (user preference for hyperlegible faces).
- Light, dark and high-contrast (`prefers-contrast: more`) variants; 11px text floor; dialogs with focus traps; 44px hit areas on the scanner.
