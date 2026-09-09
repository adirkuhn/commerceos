// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { App } from '../App'
import type { APIResponse, ThemePage } from '../types'

function mockFetch() {
  const body: APIResponse<ThemePage> = { data: { slug: 'home', canvas_width: 1440, blocks: [], custom_css: '' }, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('App', () => {
  it('renders the home route by default', async () => {
    mockFetch()
    render(<App />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Products' })).toBeInTheDocument()
  })

  it('visiting /store/:key remembers the key and redirects to home -- the link the admin app shows for a channel', async () => {
    mockFetch()
    localStorage.clear()
    window.history.pushState({}, '', '/store/channel-abc-123')

    render(<App />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
    expect(window.location.pathname).toBe('/')
    expect(localStorage.getItem('commerceos_storefront_key')).toBe('channel-abc-123')
  })
})
