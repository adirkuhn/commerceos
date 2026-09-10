import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type ColorMode = 'light' | 'dark' | 'system'

interface UIState {
  isMobileNavOpen: boolean
  openMobileNav: () => void
  closeMobileNav: () => void
  themeId: string
  setThemeId: (id: string) => void
  colorMode: ColorMode
  setColorMode: (mode: ColorMode) => void
  locale: string
  setLocale: (locale: string) => void
  currency: string
  setCurrency: (currency: string) => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      isMobileNavOpen: false,
      openMobileNav: () => set({ isMobileNavOpen: true }),
      closeMobileNav: () => set({ isMobileNavOpen: false }),
      themeId: 'aurora',
      setThemeId: (id) => set({ themeId: id }),
      colorMode: 'system',
      setColorMode: (mode) => set({ colorMode: mode }),
      locale: 'en-US',
      setLocale: (locale) => set({ locale }),
      currency: 'USD',
      setCurrency: (currency) => set({ currency }),
    }),
    { name: 'aurora-ui-preferences' },
  ),
)
