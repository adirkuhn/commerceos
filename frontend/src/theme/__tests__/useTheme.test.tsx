// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { THEME_STORAGE_KEY } from '../themePreference'
import { useTheme } from '../useTheme'

type Listener = (e: { matches: boolean }) => void

// jsdom has no matchMedia -- a controllable stand-in for the
// prefers-color-scheme query.
function installMatchMedia(initialDark: boolean) {
  const listeners = new Set<Listener>()
  const mql = {
    matches: initialDark,
    media: '(prefers-color-scheme: dark)',
    addEventListener: (_: string, l: Listener) => listeners.add(l),
    removeEventListener: (_: string, l: Listener) => listeners.delete(l),
  }
  window.matchMedia = (() => mql) as unknown as typeof window.matchMedia
  return {
    setDark(dark: boolean) {
      mql.matches = dark
      listeners.forEach(l => l({ matches: dark }))
    },
    listenerCount: () => listeners.size,
  }
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('dark')
})

afterEach(() => {
  localStorage.clear()
})

describe('useTheme', () => {
  it('follows the system appearance by default', () => {
    installMatchMedia(true)
    const { result } = renderHook(() => useTheme())
    expect(result.current.preference).toBe('system')
    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('reacts when the system appearance changes while on system', () => {
    const media = installMatchMedia(false)
    const { result } = renderHook(() => useTheme())
    expect(result.current.theme).toBe('light')

    act(() => media.setDark(true))

    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('cycles the preference, applies it and remembers it', () => {
    installMatchMedia(true)
    const { result } = renderHook(() => useTheme())

    act(() => result.current.cyclePreference())

    expect(result.current.preference).toBe('light')
    expect(result.current.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
  })

  it('starts from the remembered preference', () => {
    installMatchMedia(false)
    localStorage.setItem(THEME_STORAGE_KEY, 'dark')
    const { result } = renderHook(() => useTheme())
    expect(result.current.preference).toBe('dark')
    expect(result.current.theme).toBe('dark')
  })

  it('stops listening to the system on unmount', () => {
    const media = installMatchMedia(false)
    const { unmount } = renderHook(() => useTheme())
    expect(media.listenerCount()).toBe(1)
    unmount()
    expect(media.listenerCount()).toBe(0)
  })
})
