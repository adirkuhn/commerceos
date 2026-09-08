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
}))

import * as themeApi from '../../theme/api'

function renderAt(pathname: string) {
  return render(
    <MemoryRouter initialEntries={[pathname]}>
      <ThemeEditorPage />
    </MemoryRouter>,
  )
}

const emptyPage: ThemePage = { sales_channel_id: 'chan-1', slug: 'home', canvas_width: 1440, blocks: [], custom_css: '' }

const heroPage: ThemePage = {
  sales_channel_id: 'chan-1',
  slug: 'home',
  canvas_width: 1440,
  blocks: [
    { id: 'b1', type: 'hero', layout: { desktop: { x: 10, y: 10, width: 300, height: 150 }, mobile: null }, props: { heading: 'Big sale' } },
  ],
  custom_css: '',
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
})
