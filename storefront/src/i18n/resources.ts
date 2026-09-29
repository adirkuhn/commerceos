import type { Resource } from 'i18next'
import { DEFAULT_LOCALE, type Locale } from './locale'
import enShop from './locales/en/shop.json'

// Shopper texts. English (the fallback) is always bundled; each other
// language is its own small chunk, loaded before the page renders -- by
// the SSR server for the store's language, and by the browser (from
// <html lang>) before hydrating, so both render the same text. The editor
// bundles every language (editor/i18n/editorResources.ts). To add a
// language see docs/i18n.md.

/** shop: shopper texts (store language). editor: the theme editor's own
 *  texts (the admin user's language), editor bundle only. */
export const NAMESPACES = ['shop', 'editor'] as const
export type Namespace = (typeof NAMESPACES)[number]

type ShopCatalog = typeof enShop

/** Loaded shop texts per language; createI18n reads from here. */
export const resources: Partial<Record<Locale, Resource[string]>> = { [DEFAULT_LOCALE]: { shop: enShop } }

const loaders = import.meta.glob<ShopCatalog>(['./locales/*/shop.json', '!./locales/en/shop.json'], { import: 'default' })

/** Makes a language's shop texts available (no-op once loaded). */
export async function loadShopLocale(locale: Locale): Promise<void> {
  if (resources[locale]) return
  const load = loaders[`./locales/${locale}/shop.json`]
  if (load) resources[locale] = { shop: await load() }
}

/** For bundles that include every language up front (the editor). */
export function registerShopLocale(locale: Locale, shop: ShopCatalog) {
  resources[locale] = { ...resources[locale], shop }
}
