import type { Resource } from 'i18next'
import type { Locale } from '../../i18n/locale'
import { registerShopLocale } from '../../i18n/resources'
import enShop from '../../i18n/locales/en/shop.json'
import nlShop from '../../i18n/locales/nl/shop.json'
import ptBRShop from '../../i18n/locales/pt-BR/shop.json'
import enEditor from '../../i18n/locales/en/editor.json'
import nlEditor from '../../i18n/locales/nl/editor.json'
import ptBREditor from '../../i18n/locales/pt-BR/editor.json'

// The editor previews stores in any language, so it bundles every shop
// catalog (and makes them available to the canvas's createI18n).
registerShopLocale('pt-BR', ptBRShop)
registerShopLocale('nl', nlShop)

/** Shop texts plus the editor's own, for the editor bundle only. */
export const editorResources: Record<Locale, Resource[string]> = {
  en: { shop: enShop, editor: enEditor },
  'pt-BR': { shop: ptBRShop, editor: ptBREditor },
  nl: { shop: nlShop, editor: nlEditor },
}
