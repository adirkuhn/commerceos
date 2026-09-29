import { createContext, useContext, type ReactNode } from 'react'
import type { Locale } from '../../i18n/locale'

// The language this store page is in, the others the store offers, and
// where the current page lives in each (see runtime/storeLocale.ts).
export interface StoreLanguageValue {
  storeKey: string
  locale: Locale
  offered: Locale[]
  hrefFor: (locale: Locale) => string
}

const Ctx = createContext<StoreLanguageValue | null>(null)

export function StoreLanguageProvider({ value, children }: { value: StoreLanguageValue; children: ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useStoreLanguage = () => useContext(Ctx)
