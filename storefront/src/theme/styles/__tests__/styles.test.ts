import { describe, it, expect } from 'vitest'
import { styleCSS } from '..'
import { buildHead } from '../../../runtime/head'
import { defaultSettings } from '../../../core/settings'
import type { StorefrontTheme } from '../../../core/document'

describe('theme styles', () => {
  it('the default style adds nothing; supermarket dresses the components', () => {
    expect(styleCSS({ style: 'default' })).toBe('')
    expect(styleCSS({})).toBe('')
    const css = styleCSS({ style: 'supermarket' })
    expect(css).toContain('.cos-product-tile:hover { border-color: var(--color-primary); box-shadow: 0 5px 12px rgba(0, 0, 0, 0.12); }')
    expect(css).toContain('.cos-footer { background: #f0f3f2; padding: 4rem 0; }')
  })

  it('nothing animates in the supermarket style', () => {
    expect(styleCSS({ style: 'supermarket' })).toContain('transition: none !important; animation: none !important;')
  })

  it('pages carry the chosen style with the theme settings', () => {
    const theme = { version_id: 'v', version_number: 1, preview: false, documents: [], settings: { ...defaultSettings(), style: 'supermarket' } } as StorefrontTheme
    const head = buildHead({ theme, page: { route: { kind: 'template', key: 'home', notFound: false } }, preview: false, storeName: 'Shop' })
    expect(head).toContain('.cos-hero-badge')
  })
})
