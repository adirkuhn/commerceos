import { useTranslation } from 'react-i18next'
import { formatPrice } from '../core/money'
import { INTL_LOCALE, resolveLocale } from './locale'

const dateFormats = new Map<string, Intl.DateTimeFormat>()

/** Prices and dates in the store's language, the same on server and browser. */
export function useShopFormat() {
  const { i18n } = useTranslation()
  const intl = INTL_LOCALE[resolveLocale(i18n.language)]
  return {
    price: (minorUnits: number | null | undefined, currency: string | null | undefined) => formatPrice(minorUnits, currency, intl),
    date: (iso: string) => {
      const d = new Date(iso)
      if (Number.isNaN(d.getTime())) return ''
      let fmt = dateFormats.get(intl)
      if (!fmt) {
        fmt = new Intl.DateTimeFormat(intl, { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
        dateFormats.set(intl, fmt)
      }
      return fmt.format(d)
    },
  }
}
