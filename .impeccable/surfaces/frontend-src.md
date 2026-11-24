---
version: 1
slug: "frontend-src"
primary_target: "frontend/src"
related_targets: []
---

# Surface: CommerceOS admin (frontend/src)

Scope: the whole admin app (shell, navigation, every working screen). Mode: Operate. Scanner app (dark-only, phone) and storefront are out of scope.
Audience/job: owner at a desk (orders, buying, catalog, reports) and warehouse staff (pick, pack, receive) of one specialty food webshop. Money hidden by default.
Constraints: one design replaces classic and depot (switch removed); light + dark + high-contrast; en/pt-BR/nl; per-shop appearance setting keeps working; every page keeps its URL; preserve all behaviour, API calls and data flows.

## Direction contract

THESIS: Every order is a coupon book. The admin owns the idea that work moves through legs (paid, picked, packed, shipped) that tear off in sequence and never disappear: done legs carry a stamp, the current leg is raised, cancelled things are stamped VOID. It refuses the category default of grey cards, pill filters and status badges floating on a blank canvas.

OWN-WORLD: Carrier navy rail (#14284B) with white type and a jet-red marker on the active item; stock-white ground (#EEF0F3) under coupon-white panels with hairline navy-grey rules like fare boxes; carbon purple (#4B2E83 family) reserved for filled figures (quantities, totals, counts); jet red only for void, danger and the active marker. Archivo grotesk, tracked caps for tab heads, table heads and field labels; tabular figures everywhere. Perforated dashed edges with round notches only on leg coupons. Navy solid primary buttons, outlined secondary. Dark appearance is Graphite Console (user choice 2026-11-24): graphite ground #1B1D21, panels #25282D split by 1px seams #34383F instead of shadows, text #E6E6E3, muted #9A9DA3, accent blue #7FB4FF for primary and selection, red #F07A6A destructive only, green #7FD1A0 success; same layout and components. Both appearances keep destructive actions isolated by empty space at the far end of an action row.

STORY: Staff open Today and see what legs are waiting (to pick, to pack, to ship); they open an order and read where it is in its journey at a glance, act on the current leg, and move on. Lists are ruled tables with caps heads and a span line; detail pages are fare-box field grids.

FIRST VIEWPORT: Sales order detail: navy rail left; breadcrumb top bar with search; title row with order number, channel chip, status, actions right (Cancel secondary red text, primary navy). Directly under it the four leg coupons in a row, done legs stamped, current raised with progress. Below, line items table (lot chips, carbon-purple quantities, masked money) beside a customer & shipping fare-box panel.

FORM: Ticket Wallet challenger (vernacular-ephemera-jet-age-ticket-wallet), chosen by the user over the assigned Carton Mark, paired with Graphite Console (digital-design-canon-dark-first-developer-console) as the dark appearance; position: dealt challengers. seed:4d03e6da604005aa. Signature interaction: the leg coupon strip (order fulfilment, purchase-order receiving) with stamp-on-complete; VOID stamp for cancelled records. Motion: 150-200ms state transitions; a leg's stamp lands once on completion; nothing else animates.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
