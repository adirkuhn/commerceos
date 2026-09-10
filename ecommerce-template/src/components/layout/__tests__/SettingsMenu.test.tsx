// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SettingsMenu } from '../SettingsMenu'

// SettingsMenu's Radix DropdownMenu.Content is Popper-positioned, and
// actually opening it in jsdom (real PointerEvent dispatched at the
// trigger) hangs indefinitely even with ResizeObserver/IntersectionObserver
// stubbed -- a known category of Radix Popper + jsdom issue, not a bug in
// this component. The click-driven wiring (each item's onSelect calls
// useUIStore's setThemeId/setColorMode/setLocale) is verified by direct
// code path instead: stores/__tests__/uiStore.test.ts covers those actions,
// and themes/__tests__/ThemeProvider.test.tsx covers the store state
// actually reaching the DOM as CSS variables. This test only covers what's
// safe to assert here: the trigger renders correctly and the menu starts
// closed.
describe('SettingsMenu', () => {
  it('renders a closed settings trigger', () => {
    render(<SettingsMenu />)
    const trigger = screen.getByRole('button', { name: 'Theme and language settings' })
    expect(trigger).toBeInTheDocument()
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })
})
