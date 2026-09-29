// Creates the Cantinho Brasileiro storefront theme (see seed/cantinho.ts)
// through the regular APIs: collections, a theme from the "mercado"
// preset, Portuguese content, the brand colors -- and publishes it.
//
//   COMMERCEOS_TOKEN=<owner JWT> API_BASE_URL=https://…/api/v1 node scripts/seed-cantinho.ts
//   (local stack: JWT_SECRET + TENANT_ID mint a token instead of COMMERCEOS_TOKEN)
//
// Options (env): CHANNEL_ID  use this themed internal channel instead of creating one
//                NO_PUBLISH  leave the theme as a draft
// Safe to run again: existing collections (by slug) are reused; each run
// adds a new theme.
import { createHmac, randomUUID } from 'node:crypto'
import { cantinhoDocument, collectionInput, COLORS, MENU_COLLECTIONS, ROW_COLLECTIONS, STORE, type CollectionIds, type Node } from './seed/cantinho.ts'

const API = (process.env.API_BASE_URL ?? 'https://api-commerceos.dev.localhost/api/v1').replace(/\/+$/, '')

function ownerToken(): string {
  if (process.env.COMMERCEOS_TOKEN) return process.env.COMMERCEOS_TOKEN
  const secret = process.env.JWT_SECRET
  const tenantId = process.env.TENANT_ID
  if (!secret || !tenantId) throw new Error('set COMMERCEOS_TOKEN, or JWT_SECRET and TENANT_ID for a local stack')
  const b64 = (v: object) => Buffer.from(JSON.stringify(v)).toString('base64url')
  const now = Math.floor(Date.now() / 1000)
  const head = b64({ alg: 'HS256', typ: 'JWT' })
  const body = b64({ user_id: randomUUID(), tenant_id: tenantId, role: 'customer_owner', iat: now, exp: now + 3600 })
  return `${head}.${body}.${createHmac('sha256', secret).update(`${head}.${body}`).digest('base64url')}`
}

async function call<T>(token: string, method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    method, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const json = (await res.json().catch(() => ({}))) as { data?: T; error?: { message?: string } }
  if (!res.ok) throw new Error(`${method} ${path}: ${res.status} ${json.error?.message ?? ''}`)
  return json.data as T
}

interface Channel { id: string; name: string; type: string; storefront_mode?: string; storefront_locale?: string }

async function channel(token: string): Promise<Channel> {
  if (process.env.CHANNEL_ID) {
    const all = await call<Channel[]>(token, 'GET', '/integrations/sales-channels')
    const ch = all.find(c => c.id === process.env.CHANNEL_ID)
    if (!ch || ch.type !== 'internal' || ch.storefront_mode !== 'theme') throw new Error('CHANNEL_ID must be an internal channel with a hosted storefront')
    if (ch.storefront_locale !== STORE.locale) console.warn(`! set the channel's Storefront language to ${STORE.locale} in admin (it is ${ch.storefront_locale ?? 'en'})`)
    return ch
  }
  const [warehouse] = await call<{ id: string }[]>(token, 'GET', '/inventory/warehouses')
  if (!warehouse) throw new Error('create a warehouse first')
  return call<Channel>(token, 'POST', '/integrations/sales-channels', {
    name: STORE.name, type: 'internal', enrollment_mode: 'all', storefront_mode: 'theme',
    storefront_locale: STORE.locale, default_warehouse_id: warehouse.id,
  })
}

async function collections(token: string): Promise<CollectionIds> {
  const categories = await call<{ id: string; name: string }[]>(token, 'GET', '/pms/categories')
  const existing = await call<{ id: string; slug: string }[]>(token, 'GET', '/collections?per_page=100')
  const ids: CollectionIds = {}
  const specs = [...MENU_COLLECTIONS, ...ROW_COLLECTIONS].filter((s, i, all) => all.findIndex(o => o.slug === s.slug) === i)
  for (const spec of specs) {
    const found = existing.find(c => c.slug === spec.slug)
    const col = found ?? await call<{ id: string; slug: string }>(token, 'POST', '/collections', collectionInput(spec, categories))
    ids[spec.slug] = { id: col.id, slug: col.slug }
    console.log(`${found ? '=' : '+'} collection ${spec.slug}`)
  }
  return ids
}

interface Draft { version: { revision: number; settings: Record<string, unknown> }; documents: { kind: string; key: string }[] }

async function main() {
  const token = ownerToken()
  const ch = await channel(token)
  console.log(`channel ${ch.name} (${ch.id})`)
  const ids = await collections(token)

  const theme = await call<{ id: string }>(token, 'POST', '/theme/themes', { sales_channel_id: ch.id, name: STORE.name, preset_key: 'mercado' })
  const session = await call<{ token: string }>(token, 'POST', `/theme/themes/${theme.id}/sessions`, { purpose: 'edit', locale: STORE.locale })
  const editor = session.token

  const draft = await call<Draft>(editor, 'POST', '/theme-editor/draft')
  let revision = (await call<{ revision: number }>(editor, 'PUT', '/theme-editor/draft/settings', {
    expected_revision: draft.version.revision, settings: { ...draft.version.settings, colors: COLORS },
  })).revision

  for (const d of draft.documents) {
    const doc = await call<{ root: Node; title?: string }>(editor, 'GET', `/theme-editor/draft/documents/${d.kind}/${d.key}`)
    const root = cantinhoDocument(d.kind, d.key, doc.root, ids)
    revision = (await call<{ revision: number }>(editor, 'PUT', `/theme-editor/draft/documents/${d.kind}/${d.key}`, {
      expected_revision: revision, root, ...(d.kind === 'page' ? { title: doc.title } : {}),
    })).revision
    console.log(`✓ ${d.kind}/${d.key}`)
  }

  if (!process.env.NO_PUBLISH) {
    const v = await call<{ number: number }>(editor, 'POST', '/theme-editor/publish', { expected_revision: revision })
    console.log(`published version ${v.number}`)
  }
  console.log(`\nStore: ${new URL(API).origin}/store/${ch.id}/`)
}

main().catch(err => {
  console.error(err instanceof Error ? err.message : err)
  process.exit(1)
})
