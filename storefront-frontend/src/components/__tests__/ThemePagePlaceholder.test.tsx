// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { ThemePagePlaceholder } from '../ThemePagePlaceholder'
import type { APIResponse, ThemePage } from '../../types'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

function mockFetchError(message: string) {
  const body: APIResponse<null> = { data: null, error: { code: 'INTERNAL', message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('ThemePagePlaceholder', () => {
  it('shows the block count once loaded', async () => {
    const page: ThemePage = {
      slug: 'home', canvas_width: 1440,
      blocks: [{ id: 'b1', type: 'hero', layout: { desktop: { x: 0, y: 0, width: 1, height: 1 }, mobile: null }, props: {} }],
    }
    mockFetch(page)

    render(<ThemePagePlaceholder slug="home" title="Home" />)

    expect(await screen.findByText(/1 block configured/)).toBeInTheDocument()
  })

  it('shows an empty-page message when there are no blocks', async () => {
    mockFetch<ThemePage>({ slug: 'home', canvas_width: 1440, blocks: [] })

    render(<ThemePagePlaceholder slug="home" title="Home" />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
  })

  it('shows an error message on failure', async () => {
    mockFetchError('boom')

    render(<ThemePagePlaceholder slug="home" title="Home" />)

    await waitFor(() => expect(screen.getByText(/Could not load this page: boom/)).toBeInTheDocument())
  })
})
