// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

beforeEach(() => {
  document.documentElement.className = ''
  document.head.innerHTML = ''
  vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() }))
})

function openWith(extra: Partial<Draft> = {}) {
  const api = new EditorAPI({ baseURL: 'x', token: 't' })
  api.openDraft = vi.fn(async () => ({ ...draft, ...extra }))
  return api
}

describe('EditorApp side panel', () => {
  it('shows the components to add, with the layers one tab away', async () => {
    render(<EditorApp api={openWith()} apiBase="https://api.test" />)
    await screen.findByText('Autumn')

    expect(screen.getByRole('tab', { name: 'Add' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByTestId('editor-blocks')).toBeVisible()
    expect(screen.getByTestId('editor-layers')).not.toBeVisible()

    await userEvent.click(screen.getByRole('tab', { name: 'Layers' }))

    expect(screen.getByTestId('editor-layers')).toBeVisible()
    expect(screen.getByTestId('editor-blocks')).not.toBeVisible()
  })
})

describe('EditorApp look', () => {
  it('opens dark when the admin was dark', async () => {
    render(<EditorApp api={openWith()} apiBase="https://api.test" themePreference="dark" />)
    await screen.findByText('Autumn')

    expect(document.documentElement).toHaveClass('dark')
  })

  it('uses the shop appearance for its own controls', async () => {
    const appearance = { font: 'inter', accent_color: '#0B6E4F', sidebar_color: '#14163F', corners: 'normal', text_size: 'normal', density: 'comfortable' } as const
    render(<EditorApp api={openWith({ appearance })} apiBase="https://api.test" />)
    await screen.findByText('Autumn')

    await waitFor(() => expect(document.getElementById('editor-appearance')?.textContent).toContain('--ed-accent: 11 110 79;'))
    expect(document.getElementById('editor-font')?.getAttribute('href')).toContain('family=Inter')
  })
})

describe('EditorApp', () => {
  // The editor used to render its layout bare until the draft (and so the
  // store) was known, then inside a context provider: React remounted the
  // layout and the canvas GrapesJS lived in was thrown away -- a blank
  // editor with no components.
  it('keeps the nodes GrapesJS mounts into when the draft loads', async () => {
    handed.length = 0
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

