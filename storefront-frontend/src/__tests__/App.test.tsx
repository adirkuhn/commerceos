// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { App } from '../App'
import type { APIResponse, ThemePage } from '../types'

function mockFetch() {
  const body: APIResponse<ThemePage> = { data: { slug: 'home', canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'freeform' }, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

function pageFor(slug: string, blocks: ThemePage['blocks']): ThemePage {
  return { slug: slug as ThemePage['slug'], canvas_width: 1440, blocks, custom_css: '', editor_mode: 'freeform' }
}

// Unlike mockFetch() (always the same 'home' body, fine for routing-only
// tests), this responds per-slug from the requested URL -- needed to
// prove Layout's header/footer (see App.tsx's own Layout doc comment)
// actually render their own distinct content, not just whatever the home
// page happened to return.
function mockFetchPerSlug(bySlug: Record<string, ThemePage>) {
  vi.stubGlobal('fetch', vi.fn((url: string) => {
    const slug = url.match(/\/theme\/pages\/([\w-]+)/)?.[1] ?? 'home'
    const page = bySlug[slug] ?? pageFor(slug, [])
    const body: APIResponse<ThemePage> = { data: page, error: null }
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body) })
  }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('App', () => {
  it('renders the home route by default', async () => {
    mockFetch()
    render(<App />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
  })

  it('renders header (nav_actions + nav_menu) and footer content around every page, from their own pseudo-pages', async () => {
    mockFetchPerSlug({
      home: pageFor('home', []),
      header: pageFor('header', [
        { id: 'nav1', type: 'nav_menu', layout: { desktop: { x: 0, y: 0, width: 200, height: 40 }, mobile: null }, props: { links: [{ label: 'Products', url: '/products' }] } },
      ]),
      footer: pageFor('footer', [
        { id: 'ft1', type: 'rich_text', layout: { desktop: { x: 0, y: 0, width: 400, height: 40 }, mobile: null }, props: { content: 'Copyright 2026' } },
      ]),
    })

    render(<App />)

    // nav_actions is header's required block -- synthesized even though
    // this mock's header page never explicitly includes it.
    expect(await screen.findByRole('link', { name: 'Cart' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Products' })).toHaveAttribute('href', '/products')
    expect(screen.getByText('Copyright 2026')).toBeInTheDocument()
  })

  it('visiting /enter/:key remembers the key and redirects to home -- the link the admin app shows for a channel', async () => {
    mockFetch()
    localStorage.clear()
    window.history.pushState({}, '', '/enter/channel-abc-123')

    render(<App />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
    expect(window.location.pathname).toBe('/')
    expect(localStorage.getItem('commerceos_storefront_key')).toBe('channel-abc-123')
  })

  it('honors ?next= to land on a specific page -- the theme editor\'s Preview button', async () => {
    mockFetch()
    localStorage.clear()
    window.history.pushState({}, '', '/enter/channel-abc-123?next=/products')

    render(<App />)

    // ProductsPage is theme-driven too (CanvasPage), so this same "no
    // content yet" text just confirms rendering settled past the
    // redirect -- the real assertion is the URL below.
    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
    expect(window.location.pathname).toBe('/products')
  })

  it('ignores a protocol-relative ?next= (off-site redirect attempt) and falls back to home', async () => {
    mockFetch()
    localStorage.clear()
    window.history.pushState({}, '', '/enter/channel-abc-123?next=//evil.example.com')

    render(<App />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
    expect(window.location.pathname).toBe('/')
  })
})
