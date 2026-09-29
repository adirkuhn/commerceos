import 'i18next'
import type enEditor from './locales/en/editor.json'
import type enShop from './locales/en/shop.json'

// Typed keys: t('cart.empty') is checked against the English catalogs.
declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'shop'
    resources: { shop: typeof enShop; editor: typeof enEditor }
  }
}
