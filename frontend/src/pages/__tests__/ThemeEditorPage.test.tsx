// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import ThemeEditorPage from '../ThemeEditorPage'
import type { ThemePage } from '../../theme/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../theme/api', () => ({
  getThemePage: vi.fn(),
  saveThemePage: vi.fn(),
  createEditSession: vi.fn(),
}))

import * as themeApi from '../../theme/api'

function renderAt(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <ThemeEditorPage />
    </MemoryRouter>,
  )
}

const emptyPage: ThemePage = {
  sales_channel_id: 'chan-1', slug: 'home', canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'puck', puck_data: { content: [], root: {} },
}

const heroBlock = { id: 'b1', type: 'hero' as const, layout: { desktop: { x: 10, y: 10, width: 300, height: 150 }, mobile: null }, props: { heading: 'Big sale' } }

const freeformPageWithContent: ThemePage = {
  sales_channel_id: 'chan-1', slug: 'home', canvas_width: 1440, blocks: [heroBlock], custom_css: '', editor_mode: 'freeform',
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ThemeEditorPage', () => {
  it('shows a message for a URL that does not match the theme editor pattern', () => {
    renderAt('/sales-channels/chan-1/theme/not-a-real-page')

    expect(screen.getByText(/isn't valid/)).toBeInTheDocument()
  })

  it('loads and shows the current page slug', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')

    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalledWith('test-token', 'chan-1', 'home'))
    expect(screen.getByLabelText('Page')).toHaveValue('home')
  })

  it('navigates to a different page slug via the picker', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    await user.selectOptions(screen.getByLabelText('Page'), 'footer')

    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalledWith('test-token', 'chan-1', 'footer'))
  })

  it('auto-migrates a freeform page with real content to Puck on open', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(freeformPageWithContent)
    vi.mocked(themeApi.saveThemePage).mockResolvedValue({ ...freeformPageWithContent, editor_mode: 'puck' })

    renderAt('/sales-channels/chan-1/theme/home')

    await waitFor(() => expect(themeApi.saveThemePage).toHaveBeenCalledWith('test-token', 'chan-1', 'home', expect.objectContaining({
      editor_mode: 'puck',
      puck_data: expect.objectContaining({ content: expect.arrayContaining([expect.objectContaining({ type: 'Hero' })]) }),
    })))
  })

  it('does not touch a page already on Puck', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    expect(themeApi.saveThemePage).not.toHaveBeenCalled()
  })

  it('does not touch a freeform page with no real content (nothing to convert)', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue({ ...emptyPage, editor_mode: 'freeform', puck_data: undefined })

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    expect(themeApi.saveThemePage).not.toHaveBeenCalled()
  })

  it('"Edit visually" mints an edit session and opens storefront-frontend\'s Puck editor in a new tab', async () => {
    const user = userEvent.setup()
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.test')
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)
    vi.mocked(themeApi.createEditSession).mockResolvedValue({ token: 'edit-token-1', expires_at: '2026-01-01T00:00:00Z' })
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /Edit visually/ }))

    await waitFor(() => expect(themeApi.createEditSession).toHaveBeenCalledWith('test-token', 'chan-1'))
    expect(openSpy).toHaveBeenCalledWith('https://shop.test/store/edit/home?token=edit-token-1', '_blank', 'noopener,noreferrer')
  })

  it('shows an error when starting a visual edit session fails', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)
    vi.mocked(themeApi.createEditSession).mockRejectedValue(new Error('session limit reached'))

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /Edit visually/ }))

    expect(await screen.findByText('session limit reached')).toBeInTheDocument()
  })

  it('shows a working "Preview" link for a slug with a real storefront route', async () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.test')
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    const link = screen.getByRole('link', { name: /Preview/ })
    expect(link).toHaveAttribute('href', 'https://shop.test/store/enter/chan-1?next=%2F')
  })

  it('disables "Preview" for a slug with no fixed storefront route (order-confirmation)', async () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.test')
    vi.mocked(themeApi.getThemePage).mockResolvedValue({ ...emptyPage, slug: 'order-confirmation' })

    renderAt('/sales-channels/chan-1/theme/order-confirmation')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    expect(screen.queryByRole('link', { name: /Preview/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Preview/ })).toBeDisabled()
  })

  it('edits and saves custom CSS, keeping editor_mode puck and round-tripping puck_data unchanged', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)
    vi.mocked(themeApi.saveThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: 'Custom CSS' }))
    await user.click(screen.getByLabelText('Custom CSS'))
    await user.paste('.hero { color: red; }')
    await user.click(screen.getByRole('button', { name: /save/i }))

    await waitFor(() => expect(themeApi.saveThemePage).toHaveBeenCalledTimes(1))
    expect(themeApi.saveThemePage).toHaveBeenCalledWith('test-token', 'chan-1', 'home', {
      canvas_width: 1440, blocks: [], custom_css: '.hero { color: red; }', editor_mode: 'puck', puck_data: emptyPage.puck_data,
    })
  })

  it('disables Save until custom CSS changes', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
  })

  it('toggles the Custom CSS panel open and closed', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    expect(screen.queryByLabelText('Custom CSS')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Custom CSS' }))
    expect(screen.getByLabelText('Custom CSS')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Custom CSS' }))
    expect(screen.queryByLabelText('Custom CSS')).not.toBeInTheDocument()
  })
})
