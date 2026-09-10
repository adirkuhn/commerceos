import { describe, expect, it, beforeEach } from 'vitest'
import { useUIStore } from '../uiStore'

beforeEach(() => useUIStore.setState({ themeId: 'aurora', colorMode: 'system', locale: 'en-US', currency: 'USD', isMobileNavOpen: false }))

describe('useUIStore', () => {
  it('sets the active theme id', () => {
    useUIStore.getState().setThemeId('meridian')
    expect(useUIStore.getState().themeId).toBe('meridian')
  })

  it('sets the color mode', () => {
    useUIStore.getState().setColorMode('dark')
    expect(useUIStore.getState().colorMode).toBe('dark')
    useUIStore.getState().setColorMode('light')
    expect(useUIStore.getState().colorMode).toBe('light')
  })

  it('sets the locale', () => {
    useUIStore.getState().setLocale('es-ES')
    expect(useUIStore.getState().locale).toBe('es-ES')
  })

  it('opens and closes the mobile nav', () => {
    useUIStore.getState().openMobileNav()
    expect(useUIStore.getState().isMobileNavOpen).toBe(true)
    useUIStore.getState().closeMobileNav()
    expect(useUIStore.getState().isMobileNavOpen).toBe(false)
  })
})
