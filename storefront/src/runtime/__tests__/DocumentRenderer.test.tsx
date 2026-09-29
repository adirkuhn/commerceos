// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import { DocumentRenderer } from '../DocumentRenderer'
import { RenderProvider } from '../RenderContext'
import type { ThemeNode } from '../../core/document'

const tree: ThemeNode = {
  id: 'root', type: 'root', children: [{
    id: 's1', type: 'section', props: { full_width: true }, style: { base: { padding_y: 'xl' } },
    children: [
      { id: 'h1', type: 'heading', props: { text: 'Welcome', level: 1 } },
      { id: 'x1', type: 'retired-widget' },
    ],
  }],
}

describe('DocumentRenderer', () => {
  it('renders the tree through the registry with node classes and styles', () => {
    const { container } = render(<DocumentRenderer root={tree} />)
    expect(container.querySelector('section.cos-section.n-s1 > h1.n-h1')?.textContent).toBe('Welcome')
    expect(container.querySelector('style')?.textContent).toBe('.n-s1{padding-top:var(--space-xl);padding-bottom:var(--space-xl)}')
  })

  it('skips components it does not know instead of failing the page', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { container } = render(<DocumentRenderer root={tree} />)
    expect(container.querySelector('[data-node="x1"]')).toBeNull()
    warn.mockRestore()
  })

  it('renders nothing for an empty document', () => {
    const { container } = render(<DocumentRenderer root={{ id: 'root', type: 'root' }} />)
    expect(container.textContent).toBe('')
  })

  it('shows a node\'s translation in the language being read, the default text otherwise', () => {
    const translated: ThemeNode = { id: 'root', type: 'root', children: [{ id: 's', type: 'section', children: [
      { id: 'h', type: 'heading', props: { text: 'Bem-vindo', level: 2 }, translations: { en: { text: 'Welcome' } } },
    ] }] }
    const inEnglish = render(<RenderProvider value={{ contentLocale: 'en' }}><DocumentRenderer root={translated} /></RenderProvider>)
    expect(inEnglish.container.querySelector('h2')?.textContent).toBe('Welcome')
    inEnglish.unmount()
    const inDutch = render(<RenderProvider value={{ contentLocale: 'nl' }}><DocumentRenderer root={translated} /></RenderProvider>)
    expect(inDutch.container.querySelector('h2')?.textContent).toBe('Bem-vindo')
  })
})
