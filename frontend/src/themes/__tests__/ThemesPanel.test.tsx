// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ThemesPanel } from '../ThemesPanel'
import type { SalesChannel } from '../../integrations/types'
import type { Theme, VersionSummary } from '../types'
import { renderAt } from '../../test/renderAt'
import { acceptConfirm } from '../../test/confirm'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({
  listThemes: vi.fn(), createTheme: vi.fn(), deleteTheme: vi.fn(), createThemeSession: vi.fn(),
  listVersions: vi.fn(), rollbackTheme: vi.fn(),
}))
import * as api from '../api'

const channel: SalesChannel = {
  id: 'chan-1', name: 'My Shop', type: 'internal', enrollment_mode: 'all', storefront_mode: 'theme', active: true, created_at: '', updated_at: '',
}

const theme = (o: Partial<Theme>): Theme => ({
  id: 't1', sales_channel_id: 'chan-1', name: 'Autumn', preset_key: 'default', is_live: false,
  created_at: '2026-09-28T10:00:00Z', updated_at: '2026-09-28T10:00:00Z', ...o,
})

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listThemes).mockResolvedValue({ items: [theme({ id: 'live', name: 'Summer', is_live: true, published_version_id: 'v1' }), theme({})], total: 2 })
})

const renderPanel = (url = '/themes') => renderAt(<ThemesPanel channel={channel} onClose={vi.fn()} />, '/themes', url)

const version = (o: Partial<VersionSummary>): VersionSummary => ({
  id: 'v1', number: 1, state: 'sealed', revision: 1, note: '', created_by: 'u', created_at: '2026-09-20T10:00:00Z', sealed_at: '2026-09-20T10:00:00Z', ...o,
})

describe('ThemesPanel', () => {
  it('lists themes with their state and links to the live store', async () => {
    renderPanel()
    const list = await screen.findByRole('list', { name: 'Themes' })
    const [summer, autumn] = within(list).getAllByRole('listitem')
    expect(within(summer).getByText('Live')).toBeInTheDocument()
    expect(within(summer).getByRole('button', { name: 'Delete' })).toBeDisabled()
    expect(within(autumn).getByText('Draft only')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /View store/ })).toHaveAttribute('href', 'https://api.commerceos.dev.localhost/store/chan-1/')
  })

  it('creates a theme', async () => {
    vi.mocked(api.createTheme).mockResolvedValue(theme({ id: 't2', name: 'Winter' }))
    renderPanel()
    await screen.findByRole('list', { name: 'Themes' })

    await userEvent.type(screen.getByLabelText('New theme name'), 'Winter')
    await userEvent.click(screen.getByRole('button', { name: /Create theme/ }))

    await waitFor(() => expect(api.createTheme).toHaveBeenCalledWith('tok', 'chan-1', 'Winter'))
    expect(api.listThemes).toHaveBeenCalledTimes(2)
  })

  it('opens the editor with a fresh edit session in the URL fragment', async () => {
    const win = { opener: {}, location: { href: '' }, close: vi.fn() }
    vi.spyOn(window, 'open').mockReturnValue(win as unknown as Window)
    vi.mocked(api.createThemeSession).mockResolvedValue({ token: 'sess', expires_at: '' })
    renderPanel()
    const list = await screen.findByRole('list', { name: 'Themes' })

    await userEvent.click(within(within(list).getAllByRole('listitem')[1]).getByRole('button', { name: /Customize/ }))

    await waitFor(() => expect(win.location.href).toBe('https://api.commerceos.dev.localhost/store/editor#session=sess&lang=en'))
    expect(api.createThemeSession).toHaveBeenCalledWith('tok', 't1', 'edit')
    expect(win.opener).toBeNull()
  })

  it('closes the blank tab and shows the error when the session cannot be created', async () => {
    const win = { opener: {}, location: { href: '' }, close: vi.fn() }
    vi.spyOn(window, 'open').mockReturnValue(win as unknown as Window)
    vi.mocked(api.createThemeSession).mockRejectedValue(new Error('this sales channel does not host a themed storefront'))
    renderPanel()
    const list = await screen.findByRole('list', { name: 'Themes' })

    await userEvent.click(within(within(list).getAllByRole('listitem')[1]).getByRole('button', { name: /Customize/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent('does not host a themed storefront')
    expect(win.close).toHaveBeenCalled()
  })

  it('deletes a theme after confirmation', async () => {
    vi.mocked(api.deleteTheme).mockResolvedValue()
    renderPanel()
    const list = await screen.findByRole('list', { name: 'Themes' })

    await userEvent.click(within(within(list).getAllByRole('listitem')[1]).getByRole('button', { name: 'Delete' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete theme' }))

    await waitFor(() => expect(api.deleteTheme).toHaveBeenCalledWith('tok', 't1'))
    expect(screen.queryByText('Autumn')).not.toBeInTheDocument()
  })

  it('explains that a headless channel must switch to a hosted storefront first', async () => {
    renderAt(<ThemesPanel channel={{ ...channel, storefront_mode: 'headless' }} onClose={vi.fn()} />, '/themes')
    expect(await screen.findByText(/This channel is headless/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Create theme/ })).toBeDisabled()
  })
})

describe('version history', () => {
  beforeEach(() => {
    vi.mocked(api.listVersions).mockResolvedValue([
      version({ id: 'draft', number: 4, state: 'draft', sealed_at: null }),
      version({ id: 'v3', number: 3 }), version({ id: 'v2', number: 2 }), version({ id: 'v1', number: 1 }),
    ])
    vi.mocked(api.listThemes).mockResolvedValue({ items: [theme({ id: 'live', name: 'Summer', is_live: true, published_version_id: 'v3' }), theme({})], total: 2 })
  })

  it('opens from the URL and lists published versions with the live one marked', async () => {
    renderPanel('/themes?history=live')
    const list = await screen.findByRole('list', { name: 'Versions of Summer' })
    const rows = within(list).getAllByRole('listitem')
    expect(rows.map(r => within(r).getByText(/^Version/).textContent)).toEqual(['Version 3', 'Version 2', 'Version 1'])
    expect(within(rows[0]).getByText('Live')).toBeInTheDocument()
    expect(within(rows[1]).getByRole('button', { name: 'Make live' })).toBeInTheDocument()
    expect(api.listVersions).toHaveBeenCalledWith('tok', 'live')
  })

  it('offers versions only for themes that were published', async () => {
    renderPanel()
    const list = await screen.findByRole('list', { name: 'Themes' })
    const [summer, autumn] = within(list).getAllByRole('listitem')
    expect(within(summer).getByRole('button', { name: /Versions/ })).toHaveAttribute('aria-expanded', 'false')
    expect(within(autumn).queryByRole('button', { name: /Versions/ })).toBeNull()
  })

  it('rolls back to an older version after confirmation', async () => {
    vi.mocked(api.rollbackTheme).mockResolvedValue(theme({ id: 'live', is_live: true, published_version_id: 'v2' }))
    const { router } = renderPanel()
    await userEvent.click(await screen.findByRole('button', { name: /Versions/ }))
    expect(router.state.location.search).toBe('?history=live')

    const list = await screen.findByRole('list', { name: 'Versions of Summer' })
    await userEvent.click(within(within(list).getAllByRole('listitem')[1]).getByRole('button', { name: 'Make live' }))
    await acceptConfirm()

    await waitFor(() => expect(api.rollbackTheme).toHaveBeenCalledWith('tok', 'live', 'v2'))
    expect(api.listThemes).toHaveBeenCalledTimes(2)
  })

  it('shows why a rollback failed', async () => {
    vi.mocked(api.rollbackTheme).mockRejectedValue(new Error('version not sealed'))
    renderPanel('/themes?history=live')
    const list = await screen.findByRole('list', { name: 'Versions of Summer' })
    await userEvent.click(within(within(list).getAllByRole('listitem')[1]).getByRole('button', { name: 'Make live' }))
    await acceptConfirm()
    expect(await screen.findByRole('alert')).toHaveTextContent('version not sealed')
  })
})
