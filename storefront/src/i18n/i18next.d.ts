import 'i18next'
import type { sourceResources } from './resources'

// Typed keys: t('cart.empty') is checked against the English catalog.
declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'shop'
    resources: typeof sourceResources
  }
}
