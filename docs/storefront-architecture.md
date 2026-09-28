# Storefront & Theme System — Architecture

Status: approved 2026-09-28 (Phase 1). Greenfield rebuild; replaces `storefront-frontend/` (Puck),
`ecommerce-template/` (Aurora) and the old `theme` backend context (`theme_pages`, `theme_edit_sessions`).

## 1. Decisions

| # | Decision |
|---|---|
| D1 | Internal sales channels have a **storefront mode**: `headless` (REST API only, default — current behaviour) or `theme` (hosted storefront rendered from the theme system). The shopper API works in both modes; theme endpoints require `theme`. |
| D2 | Public storefront is **server-side rendered** (Node + React `renderToString`/streaming, hydrated on the client). No new frontend framework. |
| D3 | **Pages are versioned inside the theme**: one publish / one rollback covers templates, pages, header/footer and settings. |
| D4 | **Collections are a real domain** (manual + rule-based), separate bounded context. Designed and approved separately before Phase 4. |
| D5 | Editor is a separate entry point of the storefront app (GrapesJS never enters the admin or shopper bundles), **launched from the admin** via a scoped edit session. |
| D6 | Fonts: curated system/self-hosted stacks **and** optional Google Fonts (opt-in per theme). |
| D7 | Outbox events for theme created / published / rolled back / deleted. Draft saves emit no event. |
| D8 | Newsletter subscription API is added (proposed separately before Phase 5; double opt-in). |

## 2. Shape

```
             Admin (frontend/)                         storefront/  (Node SSR app)
  Sales channel → mode, Themes list,        ┌────────────────────────────────────────────────┐
  versions, publish, rollback, assets        │ server/        node:http SSR, static, caching  │
  [Customize] mints edit session ─────────►  │ src/editor     GrapesJS (editor.html, client)  │
                                             │ src/components registry: definitions + React   │
                                             │ src/runtime    routes, template resolver, data │
                                             │ src/core       doc model, validation, styles   │
                                             └────────────────────────────────────────────────┘
                                                        │ JSON, OpenAPI contracts
                                  backend: theme context (new) · storefront API (kept) · collections (new)
```

- `src/runtime` and `server/` must never import `src/editor` or `grapesjs` (lint rule + bundle test).
- The canvas and the runtime share the registry and the style compiler → what the merchant sees is what ships.
- Store addressing: `/store/:channelKey/...` (the channel id is already a valid publishable key). Custom domains: later.

## 3. Theme document format

GrapesJS project data is never persisted. The editor serializer maps GrapesJS ⇄ CommerceOS documents.

```jsonc
{ "schema_version": 1, "kind": "template", "key": "product",
  "root": { "id": "n1", "type": "section", "props": {},
    "style": { "base": { "padding_y": "xl" }, "tablet": {}, "mobile": { "padding_y": "md" } },
    "children": [ { "id": "n2", "type": "add-to-cart", "props": { "show_quantity": true } } ] } }
```

- Kinds: `template` (home, product, collection, search, cart, checkout, order-confirmation, login,
  account, wishlist, not-found), `page` (merchant pages, `/pages/:slug`), `section_group` (header, footer).
- Style values are a constrained vocabulary (scale keys, theme colour tokens or hex, enums) per
  breakpoint `base | tablet | mobile`. One compiler emits scoped CSS (`.n-<id>` + media queries);
  theme settings become CSS custom properties.
- No merchant HTML, CSS or JS. Rich text is a small JSON AST (paragraph, heading, list, bold, italic, link)
  rendered by React.
- Server validates every document against the OpenAPI contract (node schema discriminated by `type`)
  plus structural rules (allowed parents/children, template data context, required components).

### Settings (per version)

brand (logo, favicon asset ids) · colors (primary, secondary, background, surface, text, accent) ·
typography (heading/body font, source `system|google`, scale) · layout (max width, gutter) ·
buttons (radius, size, style) · product (grid columns, image ratio) · header (layout, sticky) · footer (layout).

## 4. Component registry

```ts
interface ComponentDefinition<P> {
  type: string; name: string; category: ComponentCategory; icon: IconName; description: string;
  props: PropSchema<P>; defaults: P;
  allowedChildren: string[] | 'none' | 'any-content'; allowedParents: string[] | 'any';
  dataContext?: 'product' | 'collection' | 'cart' | 'customer' | 'order';
  styleGroups: StyleGroup[];
  editor: { removable: boolean; copyable: boolean; draggable: boolean; requiredIn?: TemplateKey[] };
}
```

Renderers are registered separately (`type → lazy React component`) so `core` stays React-free and
testable. A contract test keeps the registry and the OpenAPI node schema in sync.

## 5. Persistence (migration 186+)

```
sales_channels.storefront_mode  'headless' | 'theme'  (default 'headless')
themes           id, tenant_id, sales_channel_id, name, preset_key, live_version_id, is_live
                 partial UNIQUE (sales_channel_id) WHERE is_live
theme_versions   id, tenant_id, theme_id, number, state 'draft'|'sealed', revision, settings jsonb,
                 note, created_by, created_at, sealed_at
                 UNIQUE (theme_id, number); partial UNIQUE (theme_id) WHERE state = 'draft'
theme_documents  id, tenant_id, version_id, kind, key, title, seo jsonb, tree jsonb
                 UNIQUE (version_id, kind, key)
theme_assets     id, tenant_id, url, storage_key, mime, bytes, width, height, alt, created_by, created_at, deleted_at
theme_sessions   id, tenant_id, theme_id, user_id, role, purpose 'edit'|'preview', token_hash, expires_at, revoked_at
```

All tables: `tenant_id`, NULLIF RLS policy, `FORCE ROW LEVEL SECURITY`, access only via `tenant.WithTenant`.
Sealed versions/documents are immutable (domain `ErrVersionSealed` + DB trigger).

- **Publish** (one tx): lock draft, check `revision`, seal, set live version + live theme, outbox
  `storefront.theme.published`. Failure → rollback; previous version stays live.
- **Rollback** (one tx): point live version at an older sealed version, outbox `storefront.theme.rolled_back`.
- **Draft**: created lazily from the live version on first edit; every write carries `revision` (409 on conflict).

## 6. API (all OpenAPI-first, validated, `{data, error}` envelope, paginated lists)

Admin (JWT; writes `owner|admin`):
`GET/POST /api/v1/themes?sales_channel_id=` · `GET/PATCH/DELETE /api/v1/themes/{id}` ·
`GET /api/v1/themes/{id}/versions` · `POST /api/v1/themes/{id}/publish` · `POST /api/v1/themes/{id}/rollback` ·
`POST /api/v1/themes/{id}/edit-sessions` · `/api/v1/theme-assets` CRUD (multipart upload).

Editor (edit-session bearer, token hashed at rest, bound to user + theme + role):
`GET /api/v1/theme-editor/draft` · `PUT …/draft/settings` · `GET/PUT/DELETE …/draft/documents/{kind}/{key}` ·
`POST …/draft/documents` · `POST /api/v1/theme-editor/publish` · `POST …/preview-sessions` · assets.

Public (storefront key; `?preview=<token>` reads the draft, never touches live):
`GET /storefront/v1/theme` (live index + settings, ETag) ·
`GET /storefront/v1/theme/versions/{vid}/documents/{kind}/{key}` (immutable, long cache).

Missing shopper APIs to add: product by slug, category by slug, store info, collections (D4),
newsletter subscription (D8), richer facets (later).

## 7. SSR runtime

- `server/` is a small `node:http` server: resolves `/store/:channelKey/...`, loads the live theme
  index, fetches the template document (in-memory cache keyed by immutable version id), prefetches
  template data with TanStack Query, renders React, embeds dehydrated state, serves hashed static
  assets with long cache. Dev uses Vite in middleware mode.
- Backend is called over an internal URL (`STOREFRONT_API_INTERNAL_URL`).
- Cart, checkout, account render their shell server-side and load per-shopper data client-side
  (no shopper tokens in SSR cache).

## 8. GrapesJS integration

`storageManager: false`, no default blocks/panels/code viewer; devices Desktop / Tablet 768 / Mobile 375.
Each registry definition becomes a component type + block; parent/child rules map to
`draggable`/`droppable`. Containers are native droppable DOM; leaf and commerce components mount
their real React renderer inside the canvas. Custom React inspector (props + style groups per device),
GrapesJS layer manager, undo manager, toolbar (duplicate/delete/devices/save/preview/publish),
pickers (asset, product, collection, page/link), preview-context picker for product/collection templates.

## 9. Phases

1. Discovery — done.
2. Foundation — remove obsolete code, channel mode, theme domain + persistence + API, storefront app skeleton (SSR + editor entry), registry, GrapesJS shell.
3. Editor MVP — canvas, library, layers, inspector, devices, undo/redo, save, preview, publish; Section, Container, Columns, Heading, Text, Image, Button, Spacer, Product Grid, Product Card.
4. Ecommerce — collections domain (proposal first), product/collection/cart components on real APIs.
5. Default theme — plus header/footer/navigation, newsletter (proposal first), customer components.
6. Publishing/runtime hardening — rollback UI, caching, SEO, performance budget, version retention.
