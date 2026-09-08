// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { CanvasPage } from '../CanvasPage'
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

describe('CanvasPage', () => {
  it('renders each block at its saved desktop position', async () => {
    const page: ThemePage = {
      slug: 'home',
      canvas_width: 1440,
      blocks: [
        { id: 'b1', type: 'hero', layout: { desktop: { x: 10, y: 20, width: 300, height: 150 }, mobile: null }, props: { heading: 'Welcome' } },
      ],
    }
    mockFetch(page)

    render(<CanvasPage slug="home" />)

    const heading = await screen.findByText('Welcome')
    const positioned = heading.closest('[style]') as HTMLElement
    expect(positioned.style.left).toBe('10px')
    expect(positioned.style.top).toBe('20px')
  })

  it('shows an empty-page message when there are no blocks', async () => {
    mockFetch<ThemePage>({ slug: 'home', canvas_width: 1440, blocks: [] })

    render(<CanvasPage slug="home" />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
  })

  it('shows an error message on failure', async () => {
    mockFetchError('boom')

    render(<CanvasPage slug="home" />)

    await waitFor(() => expect(screen.getByText(/Could not load this page: boom/)).toBeInTheDocument())
  })
})
