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

function mockMatchMedia(matches: boolean) {
  vi.stubGlobal('matchMedia', vi.fn().mockImplementation((query: string) => ({
    matches,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })))
}

beforeEach(() => {
  vi.restoreAllMocks()
  mockMatchMedia(false)
})

describe('CanvasPage', () => {
  it('renders each block at its saved desktop position', async () => {
    const page: ThemePage = {
      slug: 'home',
      canvas_width: 1440,
      custom_css: '', editor_mode: 'freeform',
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
    mockFetch<ThemePage>({ slug: 'home', canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'freeform' })

    render(<CanvasPage slug="home" />)

    expect(await screen.findByText(/no content yet/)).toBeInTheDocument()
  })

  it('shows an error message on failure', async () => {
    mockFetchError('boom')

    render(<CanvasPage slug="home" />)

    await waitFor(() => expect(screen.getByText(/Could not load this page: boom/)).toBeInTheDocument())
  })

  it('injects custom_css as a style tag', async () => {
    mockFetch<ThemePage>({
      slug: 'home', canvas_width: 1440, custom_css: '.hero { color: red; }', editor_mode: 'freeform',
      blocks: [{ id: 'b1', type: 'hero', layout: { desktop: { x: 0, y: 0, width: 100, height: 100 }, mobile: null }, props: {} }],
    })

    const { container } = render(<CanvasPage slug="home" />)
    await screen.findByTestId('canvas-page')

    expect(container.querySelector('style')?.textContent).toBe('.hero { color: red; }')
  })

  it('uses the mobile Rect on a mobile viewport, falling back to a computed default when unset', async () => {
    mockMatchMedia(true)
    mockFetch<ThemePage>({
      slug: 'home', canvas_width: 1440, custom_css: '', editor_mode: 'freeform',
      blocks: [
        { id: 'b1', type: 'hero', layout: { desktop: { x: 100, y: 50, width: 600, height: 200 }, mobile: null }, props: { heading: 'Welcome' } },
      ],
    })

    render(<CanvasPage slug="home" />)

    const heading = await screen.findByText('Welcome')
    const positioned = heading.closest('[style]') as HTMLElement
    expect(positioned.style.left).toBe('0px') // defaultMobileRect: full width, x reset to 0
    expect(positioned.style.top).toBe('50px') // keeps the desktop y
    expect(screen.getByTestId('canvas-page')).toHaveStyle({ width: '390px' })
  })

  it('uses a saved mobile Rect over the computed default when one exists', async () => {
    mockMatchMedia(true)
    mockFetch<ThemePage>({
      slug: 'home', canvas_width: 1440, custom_css: '', editor_mode: 'freeform',
      blocks: [
        { id: 'b1', type: 'hero', layout: { desktop: { x: 100, y: 50, width: 600, height: 200 }, mobile: { x: 5, y: 5, width: 380, height: 120 } }, props: { heading: 'Welcome' } },
      ],
    })

    render(<CanvasPage slug="home" />)

    const heading = await screen.findByText('Welcome')
    const positioned = heading.closest('[style]') as HTMLElement
    expect(positioned.style.left).toBe('5px')
    expect(positioned.style.top).toBe('5px')
  })
})
