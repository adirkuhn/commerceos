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
  sales_channel_id: 'chan-1', slug: 'home', canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'freeform',
}

const heroPage: ThemePage = {
  sales_channel_id: 'chan-1',
  slug: 'home',
  canvas_width: 1440,
  blocks: [
    { id: 'b1', type: 'hero', layout: { desktop: { x: 10, y: 10, width: 300, height: 150 }, mobile: null }, props: { heading: 'Big sale' } },
  ],
  custom_css: '',
  editor_mode: 'freeform',
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ThemeEditorPage', () => {
  it('shows a message for a URL that does not match the theme editor pattern', () => {
    renderAt('/sales-channels/chan-1/theme/not-a-real-page')

    expect(screen.getByText(/isn't valid/)).toBeInTheDocument()
  })

  it('loads and renders the page slug and its blocks', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')

    expect(await screen.findByText('Big sale')).toBeInTheDocument()
    expect(themeApi.getThemePage).toHaveBeenCalledWith('test-token', 'chan-1', 'home')
  })

  it('adds a block, edits its props, and saves', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)
    vi.mocked(themeApi.saveThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /add block/i }))
    await user.click(screen.getByRole('button', { name: 'Hero' }))

    expect(screen.getByText('Welcome')).toBeInTheDocument()

    const headingInput = screen.getByLabelText('Heading')
    await user.clear(headingInput)
    await user.type(headingInput, 'Welcome shoppers')

    expect(await screen.findByText('Welcome shoppers')).toBeInTheDocument()

    const saveButton = screen.getByRole('button', { name: /save/i })
    expect(saveButton).toBeEnabled()
    await user.click(saveButton)

    await waitFor(() => expect(themeApi.saveThemePage).toHaveBeenCalledTimes(1))
    const [, , , payload] = vi.mocked(themeApi.saveThemePage).mock.calls[0]
    expect(payload.blocks).toHaveLength(1)
    expect(payload.blocks[0].type).toBe('hero')
    expect(payload.blocks[0].props.heading).toBe('Welcome shoppers')
  })

  it('deletes a block', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    await user.click(screen.getByText('Big sale'))
    await user.click(screen.getByLabelText('Delete block'))

    expect(screen.queryByText('Big sale')).not.toBeInTheDocument()
  })

  it('disables Save until something changes', async () => {
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
  })

  it('adds a functional block full-width, with no editable props', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(emptyPage)

    renderAt('/sales-channels/chan-1/theme/cart')
    await waitFor(() => expect(themeApi.getThemePage).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /add block/i }))
    await user.click(screen.getByRole('button', { name: 'Cart summary' }))

    expect(screen.getByText('Cart summary — renders live content')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Cart summary' })).toBeInTheDocument()
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Delete block')).toBeInTheDocument()
  })

  it('edits and saves custom CSS', async () => {
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
    const [, , , payload] = vi.mocked(themeApi.saveThemePage).mock.calls[0]
    expect(payload.custom_css).toBe('.hero { color: red; }')
  })

  it('closes the block props panel when opening Custom CSS, and vice versa', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    await user.click(screen.getByText('Big sale'))
    expect(screen.getByRole('heading', { name: 'Hero' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Custom CSS' }))
    expect(screen.queryByRole('heading', { name: 'Hero' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Custom CSS')).toBeInTheDocument()
  })

  it('shows a working "Preview" link for a slug with a real storefront route', async () => {
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.test')
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

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

  it('switching to Mobile shrinks the canvas to the mobile design width', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    expect(screen.getByTestId('theme-canvas')).toHaveStyle({ width: '1440px' })

    await user.click(screen.getByRole('button', { name: 'Mobile' }))

    expect(screen.getByTestId('theme-canvas')).toHaveStyle({ width: '390px' })
    expect(screen.getByText('Big sale')).toBeInTheDocument() // falls back to a computed default, still renders
  })

  it('shows the current editor mode and switches it without needing unsaved changes', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)
    vi.mocked(themeApi.saveThemePage).mockResolvedValue({ ...heroPage, editor_mode: 'puck' })

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    expect(screen.getByRole('button', { name: 'Freeform' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Puck' })).toHaveAttribute('aria-pressed', 'false')

    await user.click(screen.getByRole('button', { name: 'Puck' }))

    await waitFor(() => expect(themeApi.saveThemePage).toHaveBeenCalledTimes(1))
    const [, , , payload] = vi.mocked(themeApi.saveThemePage).mock.calls[0]
    expect(payload.editor_mode).toBe('puck')
    expect(payload.blocks).toBe(heroPage.blocks)
    expect(payload.custom_css).toBe(heroPage.custom_css)
    await waitFor(() => expect(screen.getByRole('button', { name: 'Puck' })).toHaveAttribute('aria-pressed', 'true'))
  })

  it('does not re-save when clicking the already-active mode', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    await user.click(screen.getByRole('button', { name: 'Freeform' }))

    expect(themeApi.saveThemePage).not.toHaveBeenCalled()
  })

  it('"Edit visually" mints an edit session and opens storefront-frontend\'s Puck editor in a new tab', async () => {
    const user = userEvent.setup()
    vi.stubEnv('VITE_STOREFRONT_URL', 'https://shop.test')
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)
    vi.mocked(themeApi.createEditSession).mockResolvedValue({ token: 'edit-token-1', expires_at: '2026-01-01T00:00:00Z' })
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null)

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    await user.click(screen.getByRole('button', { name: /Edit visually/ }))

    await waitFor(() => expect(themeApi.createEditSession).toHaveBeenCalledWith('test-token', 'chan-1'))
    expect(openSpy).toHaveBeenCalledWith('https://shop.test/edit/home?token=edit-token-1', '_blank', 'noopener,noreferrer')
  })

  it('shows an error when starting a visual edit session fails', async () => {
    const user = userEvent.setup()
    vi.mocked(themeApi.getThemePage).mockResolvedValue(heroPage)
    vi.mocked(themeApi.createEditSession).mockRejectedValue(new Error('session limit reached'))

    renderAt('/sales-channels/chan-1/theme/home')
    await screen.findByText('Big sale')

    await user.click(screen.getByRole('button', { name: /Edit visually/ }))

    expect(await screen.findByText('session limit reached')).toBeInTheDocument()
  })
})
