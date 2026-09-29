import type { Resource } from 'i18next'
import type { Locale } from './locale'
import enShop from './locales/en/shop.json'
import nlShop from './locales/nl/shop.json'
import ptBRShop from './locales/pt-BR/shop.json'

// Every translation bundled with the app (they're small): no loading step,
// so the server and the browser render the same text. To add a language
// see docs/i18n.md.

export const NAMESPACES = ['shop'] as const
export type Namespace = (typeof NAMESPACES)[number]

/** English is the source of truth; its shape types every t() key. */
export const sourceResources = { shop: enShop }

export const resources: Record<Locale, Resource[string]> = {
  en: sourceResources,
  'pt-BR': { shop: ptBRShop },
  nl: { shop: nlShop },
}
