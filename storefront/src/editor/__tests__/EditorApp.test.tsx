// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import { I18nextProvider } from 'react-i18next'
import { createEditorI18n } from '../i18n/createEditorI18n'
import { EditorAPI, type Draft } from '../api'
import { defaultSettings } from '../../core/settings'

// GrapesJS itself isn't loaded: the hook records the DOM nodes it was
// handed, which is what the real editor mounts into once, for good.
const handed: HTMLElement[][] = []
vi.mock('../useGrapesEditor', () => ({
  DEVICES: [{ id: 'desktop', name: 'Desktop', width: '' }],
  useGrapesEditor: (mounts: () => { canvas: HTMLElement | null; blocks: HTMLElement | null; layers: HTMLElement | null }) => {
    const m = mounts()
    if (m.canvas && m.blocks && m.layers) handed.push([m.canvas, m.blocks, m.layers])
    return undefined
  },
}))
vi.mock('grapesjs/dist/css/grapes.min.css', () => ({}))
vi.mock('../../theme/base.css?inline', () => ({ default: '' }))

const { EditorApp } = await import('../EditorApp')

const draft: Draft = {
  theme: { id: 't1', name: 'Autumn', sales_channel_id: 'chan-1', is_live: false },
  version: { id: 'v1', number: 1, state: 'draft', revision: 1, settings: defaultSettings() } as Draft['version'],
  documents: [{ kind: 'template', key: 'home', title: 'Home' }],
  sales_channel_id: 'chan-1',
} as Draft

describe('EditorApp', () => {
  // The editor used to render its layout bare until the draft (and so the
  // store) was known, then inside a context provider: React remounted the
  // layout and the canvas GrapesJS lived in was thrown away -- a blank
  // editor with no components.
  it('keeps the nodes GrapesJS mounts into when the draft loads', async () => {
    const api = new EditorAPI({ baseURL: 'x', token: 't' })
    api.openDraft = vi.fn(async () => draft)
    const { container } = render(<EditorApp api={api} apiBase="https://api.test" />)
    await waitFor(() => expect(container.textContent).toContain('Autumn'))
    // Let the store-dependent services settle (they arrive a render later).
    await act(async () => { await new Promise(r => setTimeout(r, 20)) })

    const canvas = container.querySelector('main[aria-label="Canvas"] > div')
    expect(handed.length).toBeGreaterThan(0)
    for (const [c] of handed) expect(c).toBe(canvas)
  })

  it('speaks the language of the edit session once the draft opens', async () => {
    const api = new EditorAPI({ baseURL: 'x', token: 't' })
    api.openDraft = vi.fn(async () => ({ ...draft, locale: 'nl' }))
    const i18n = createEditorI18n('en')
    render(<I18nextProvider i18n={i18n}><EditorApp api={api} apiBase="https://api.test" /></I18nextProvider>)
    expect(await screen.findByRole('button', { name: 'Opslaan' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Publiceren' })).toBeInTheDocument()
    expect(i18n.language).toBe('nl')
    expect(document.documentElement.lang).toBe('nl')
  })
})

