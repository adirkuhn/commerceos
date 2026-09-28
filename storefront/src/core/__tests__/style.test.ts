import { describe, it, expect } from 'vitest'
import { compileDocumentStyles, nodeClass } from '../style'
import type { ThemeNode } from '../document'

const root = (children: ThemeNode[]): ThemeNode => ({ id: 'root', type: 'root', children })

describe('compileDocumentStyles', () => {
  it('emits one scoped rule per node and breakpoint, mobile last', () => {
    const css = compileDocumentStyles(root([
      { id: 's1', type: 'section', style: { base: { padding_y: 'xl', text_align: 'center' }, tablet: { padding_y: 'lg' }, mobile: { padding_y: 'md' } } },
    ]))

    expect(css).toContain('.n-s1{padding-top:var(--space-xl);padding-bottom:var(--space-xl);text-align:center}')
    expect(css).toContain('@media (max-width:1023px){.n-s1{padding-top:var(--space-lg);padding-bottom:var(--space-lg)}}')
    expect(css).toContain('@media (max-width:767px){.n-s1{padding-top:var(--space-md);padding-bottom:var(--space-md)}}')
    expect(css.indexOf('1023px')).toBeLessThan(css.indexOf('767px'))
  })

  it('walks nested children and skips unstyled nodes', () => {
    const css = compileDocumentStyles(root([
      { id: 'a', type: 'section', children: [{ id: 'b', type: 'heading', style: { base: { color: 'primary' } } }] },
    ]))
    expect(css).toBe('.n-b{color:var(--color-primary)}')
  })

  it('maps colours: tokens to variables, hex as-is, transparent', () => {
    const css = compileDocumentStyles(root([
      { id: 'x', type: 'heading', style: { base: { color: '#1a2B3c', background_color: 'transparent', border_color: 'accent' } } },
    ]))
    expect(css).toBe('.n-x{color:#1a2B3c;background-color:transparent;border-color:var(--color-accent)}')
  })

  it('drops anything outside the vocabulary (defence in depth)', () => {
    const css = compileDocumentStyles(root([
      { id: 'x', type: 'heading', style: { base: { color: 'red;}body{display:none', position: 'fixed', padding_y: '13px' } as Record<string, string> } },
    ]))
    expect(css).toBe('')
  })

  it('never emits a node id that could break out of the selector', () => {
    const css = compileDocumentStyles(root([{ id: 'x{}', type: 'heading', style: { base: { color: 'primary' } } }]))
    expect(css).toBe('')
  })

  it('resolves background images through the asset resolver', () => {
    const id = '0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e'
    const css = compileDocumentStyles(
      root([{ id: 'x', type: 'section', style: { base: { background_image: `asset:${id}`, background_size: 'cover' } } }]),
      { resolveAsset: a => (a === id ? 'https://cdn.example/a b.jpg' : undefined) },
    )
    expect(css).toBe('.n-x{background-image:url("https://cdn.example/a%20b.jpg");background-size:cover}')
  })

  it('maps layout helpers', () => {
    const css = compileDocumentStyles(root([
      { id: 'x', type: 'container', style: { base: { width: 'content', columns: '3', aspect_ratio: '16:9', display: 'hide' }, mobile: { display: 'show', columns: '1' } } },
    ]))
    expect(css).toBe(
      '.n-x{max-width:var(--layout-max-width);margin-left:auto;margin-right:auto;grid-template-columns:repeat(3,minmax(0,1fr));aspect-ratio:16/9;display:none}' +
      '@media (max-width:767px){.n-x{display:revert-layer;grid-template-columns:repeat(1,minmax(0,1fr))}}',
    )
  })
})

describe('nodeClass', () => {
  it('is the class the compiled rules target', () => {
    expect(nodeClass('s1')).toBe('n-s1')
  })
})
