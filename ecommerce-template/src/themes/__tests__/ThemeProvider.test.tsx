// @vitest-environment jsdom
import { render } from '@testing-library/react'
import { describe, expect, it, beforeEach } from 'vitest'
import { ThemeProvider } from '../ThemeProvider'
import { THEMES } from '../themes'
import { useUIStore } from '@/stores/uiStore'

beforeEach(() => {
  useUIStore.setState({ themeId: 'aurora', colorMode: 'light' })
  document.documentElement.removeAttribute('style')
  document.documentElement.classList.remove('dark')
})

describe('ThemeProvider', () => {
  it('writes the active theme light tokens onto :root', () => {
    render(<ThemeProvider><div /></ThemeProvider>)
    const aurora = THEMES.find(t => t.id === 'aurora')!
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe(aurora.colors.light.primary)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('re-renders with the dark token set and the dark class when colorMode is dark', () => {
    useUIStore.setState({ colorMode: 'dark' })
    render(<ThemeProvider><div /></ThemeProvider>)
    const aurora = THEMES.find(t => t.id === 'aurora')!
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe(aurora.colors.dark.primary)
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('switches CSS variables when the theme id changes to meridian', () => {
    useUIStore.setState({ themeId: 'meridian' })
    render(<ThemeProvider><div /></ThemeProvider>)
    const meridian = THEMES.find(t => t.id === 'meridian')!
    expect(document.documentElement.style.getPropertyValue('--primary')).toBe(meridian.colors.light.primary)
    expect(document.documentElement.style.getPropertyValue('--radius')).toBe(meridian.layout.radius)
  })
})
