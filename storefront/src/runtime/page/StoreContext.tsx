import { createContext, useContext, type ReactNode } from 'react'
import type { AssetRef, ThemeSettings } from '../../core/document'

// Store-wide facts components can use anywhere: its name and the theme's
// settings (logo) with the assets they refer to.
export interface StoreInfoValue {
  name: string
  settings: ThemeSettings
  assets: Record<string, AssetRef>
}

const Ctx = createContext<StoreInfoValue | null>(null)

export function StoreInfoProvider({ value, children }: { value: StoreInfoValue; children: ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useStoreInfo = () => useContext(Ctx)
