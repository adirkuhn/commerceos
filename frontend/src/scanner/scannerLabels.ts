// Shown on scanner screens when an order has no billing name or email --
// the admin UI uses an em dash, but the scanner spells it out for a
// glanceable handheld display.
import i18next from 'i18next'

export function unknownCustomerLabel(): string {
  return i18next.t('scanner:shared.unknownCustomer')
}
