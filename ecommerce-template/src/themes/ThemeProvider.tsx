import { useEffect, type ReactNode } from 'react'
import { THEMES, DEFAULT_THEME_ID } from './themes'
import type { ThemeColorTokens } from '@/types'
import { useUIStore } from '@/stores/uiStore'

function cssVarName(key: keyof ThemeColorTokens): string {
  return `--${key.replace(/([A-Z])/g, '-$1').toLowerCase()}`
}

function applyColorTokens(tokens: ThemeColorTokens) {
  const root = document.documentElement
  for (const key of Object.keys(tokens) as (keyof ThemeColorTokens)[]) {
    root.style.setProperty(cssVarName(key), tokens[key])
  }
}

function resolveIsDark(colorMode: 'light' | 'dark' | 'system'): boolean {
  if (colorMode === 'dark') return true
  if (colorMode === 'light') return false
  return typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches
}

// ThemeProvider owns applying the active ThemeConfig -- colors, radius,
// font -- onto :root as CSS custom properties (see themes.ts's doc
// comment: cssVarName mirrors tailwind.config.js's `hsl(var(--x))`
// tokens). No component imports a ThemeConfig directly; they read
// Tailwind classes (bg-primary, text-foreground, rounded-lg...) that
// resolve through these variables.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const themeId = useUIStore(s => s.themeId)
  const colorMode = useUIStore(s => s.colorMode)

  useEffect(() => {
    const theme = THEMES.find(t => t.id === themeId) ?? THEMES.find(t => t.id === DEFAULT_THEME_ID)!
    const isDark = resolveIsDark(colorMode)
    applyColorTokens(isDark ? theme.colors.dark : theme.colors.light)
    document.documentElement.classList.toggle('dark', isDark)
    document.documentElement.style.setProperty('--radius', theme.layout.radius)
    document.documentElement.style.setProperty('--font-sans', theme.typography.fontSans)

    if (colorMode !== 'system') return
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => {
      applyColorTokens(media.matches ? theme.colors.dark : theme.colors.light)
      document.documentElement.classList.toggle('dark', media.matches)
    }
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [themeId, colorMode])

  return <>{children}</>
}
