import type { Resource } from 'i18next'
import { LOCALES, type Locale } from './locale'

// Every translation file, found by path: locales/<lang>/<namespace>.json.
// Adding a namespace file needs no code change here (run `npm run
// i18n:types` to type its keys).
const files = import.meta.glob<{ default: Record<string, unknown> }>('./locales/*/*.json', { eager: true })

export const resources: Record<Locale, Resource[string]> = Object.fromEntries(LOCALES.map(l => [l, {}])) as Record<Locale, Resource[string]>

for (const [path, mod] of Object.entries(files)) {
  const m = /\.\/locales\/([^/]+)\/([^/]+)\.json$/.exec(path)
  if (!m || !(LOCALES as readonly string[]).includes(m[1])) continue
  resources[m[1] as Locale][m[2]] = mod.default as Resource[string][string]
}

/** Namespaces, from the English files (the source of truth). */
export const NAMESPACES = Object.keys(resources.en).sort()
