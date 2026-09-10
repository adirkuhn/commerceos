import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { DICTIONARIES, type Locale } from './dictionaries'
import { useUIStore } from '@/stores/uiStore'

interface I18nContextValue {
  locale: Locale
  currency: string
  t: (key: string) => string
  formatMoney: (minorUnits: number) => string
  formatDate: (iso: string) => string
  formatNumber: (value: number) => string
}

const I18nContext = createContext<I18nContextValue | null>(null)

export function I18nProvider({ children }: { children: ReactNode }) {
  const locale = useUIStore(s => s.locale) as Locale
  const currency = useUIStore(s => s.currency)

  const value = useMemo<I18nContextValue>(() => {
    const dict = DICTIONARIES[locale] ?? DICTIONARIES['en-US']
    return {
      locale,
      currency,
      t: (key) => dict[key] ?? key,
      formatMoney: (minorUnits) => new Intl.NumberFormat(locale, { style: 'currency', currency }).format(minorUnits / 100),
      formatDate: (iso) => new Intl.DateTimeFormat(locale, { year: 'numeric', month: 'short', day: 'numeric' }).format(new Date(iso)),
      formatNumber: (value) => new Intl.NumberFormat(locale).format(value),
    }
  }, [locale, currency])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}
