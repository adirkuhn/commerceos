import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const css = readFileSync(fileURLToPath(new URL('../base.css', import.meta.url)), 'utf8')

// The storefront's accessibility floor lives in base.css; these guard it.
describe('storefront base CSS', () => {
  it('keeps the reset below every component and theme style', () => {
    expect(css).toMatch(/@layer cos-reset, cos-components, cos-style;\s*@layer cos-reset \{\s*@tailwind base;/)
  })

  it('always shows keyboard focus and honours reduced motion', () => {
    expect(css).toContain(':focus-visible {')
    expect(css).toContain('@media (prefers-reduced-motion: reduce)')
  })

  it('gives buttons, fields and links touch-sized targets', () => {
    expect(css).toContain('.cos-button { min-height: 2.75rem;')
    expect(css).toContain('width: 100%; min-height: 2.75rem; padding: 0.5rem 0.75rem;')
    expect(css).toContain('.cos-nav a { display: inline-flex; align-items: center; min-height: 2.75rem; }')
  })

  it('draws text on colour fills with the derived readable colours', () => {
    expect(css).toContain('color: var(--color-on-primary, #fff)')
    expect(css).toContain('color: var(--color-on-accent, #fff)')
    expect(css).toContain('var(--color-accent-text, var(--color-accent))')
  })
})
