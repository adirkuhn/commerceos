import { useCallback, useEffect, useState } from 'react'
import {
  loadThemePreference,
  nextThemePreference,
  resolveTheme,
  saveThemePreference,
  type ResolvedTheme,
  type ThemePreference,
  type ThemeStorage,
} from './themePreference'

const DARK_QUERY = '(prefers-color-scheme: dark)'

function browserStorage(): ThemeStorage | null {
  try {
    return window.localStorage
  } catch {
    return null
  }
}

function systemPrefersDark(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(DARK_QUERY).matches
}

export interface ThemeState {
  preference: ThemePreference
  theme: ResolvedTheme
  cyclePreference: () => void
}

// Owns the appearance: resolves the stored preference against the OS
// setting, keeps <html class="dark"> in sync (Tailwind's darkMode: 'class')
// and follows OS changes live while the preference is "system".
export function useTheme(): ThemeState {
  const [preference, setPreference] = useState<ThemePreference>(() => loadThemePreference(browserStorage()))
  const [prefersDark, setPrefersDark] = useState(systemPrefersDark)

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return
    const mql = window.matchMedia(DARK_QUERY)
    const onChange = (e: { matches: boolean }) => setPrefersDark(e.matches)
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [])

  const theme = resolveTheme(preference, prefersDark)

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  const cyclePreference = useCallback(() => {
    setPreference(prev => {
      const next = nextThemePreference(prev)
      saveThemePreference(browserStorage(), next)
      return next
    })
  }, [])

  return { preference, theme, cyclePreference }
}
