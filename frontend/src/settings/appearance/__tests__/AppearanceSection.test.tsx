// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DEFAULT_APPEARANCE, type Appearance } from '../../../theme/appearance'
import { CodedApiError } from '../../codedRequest'

const auth = vi.hoisted(() => ({ role: 'customer_owner' }))
vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 't', user: { userId: 'u', role: auth.role }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

const look = vi.hoisted(() => ({
  appearance: null as Appearance | null,
  error: '',
  preview: vi.fn(),
  save: vi.fn(),
}))
vi.mock('../../../theme/AppearanceProvider', () => ({ useAppearance: () => look }))

const design = vi.hoisted(() => ({ design: 'depot' as 'depot' | 'classic', setDesign: vi.fn() }))
vi.mock('../../../theme/useDesign', () => ({ useDesign: () => design }))

import { AppearanceSection } from '../AppearanceSection'

const saved: Appearance = { ...DEFAULT_APPEARANCE, updated_at: '2026-10-12T10:00:00Z' }

beforeEach(() => {
  vi.clearAllMocks()
  auth.role = 'customer_owner'
  design.design = 'depot'
  look.appearance = saved
  look.error = ''
  look.save.mockImplementation(async (a: typeof DEFAULT_APPEARANCE) => ({ ...a, updated_at: '2026-10-12T11:00:00Z' }))
})

describe('AppearanceSection', () => {
  it('shows the saved appearance', () => {
    render(<AppearanceSection />)

    expect(screen.getByRole('radio', { name: 'Atkinson Hyperlegible Next' })).toBeChecked()
    expect(screen.getByLabelText('Accent colour hex')).toHaveValue('#2F33B8')
    expect(screen.getByRole('radio', { name: 'Comfortable' })).toBeChecked()
  })

  it('previews each change on the whole app before saving', async () => {
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'IBM Plex Sans' }))
    await userEvent.click(screen.getByRole('radio', { name: 'Round' }))

    expect(look.preview).toHaveBeenLastCalledWith({ ...DEFAULT_APPEARANCE, font: 'ibm_plex_sans', corners: 'round' })
    expect(look.save).not.toHaveBeenCalled()
  })

  it('saves the new appearance', async () => {
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'Larger' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save appearance' }))

    expect(look.save).toHaveBeenCalledWith({ ...DEFAULT_APPEARANCE, text_size: 'larger' })
    expect(await screen.findByRole('status')).toHaveTextContent('Saved')
  })

  it('blocks an accent too light for white button text and offers a darker shade', async () => {
    render(<AppearanceSection />)
    const hex = screen.getByLabelText('Accent colour hex')

    await userEvent.clear(hex)
    await userEvent.type(hex, '#FFB000')

    expect(screen.getByText(/too light for white button text/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save appearance' })).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: /Use #/ }))

    expect(screen.queryByText(/too light for white button text/)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save appearance' })).toBeEnabled()
  })

  it('does not preview a half-typed colour', async () => {
    render(<AppearanceSection />)
    const hex = screen.getByLabelText('Sidebar colour hex')

    await userEvent.clear(hex)
    await userEvent.type(hex, '#12')

    expect(look.preview).not.toHaveBeenCalledWith(expect.objectContaining({ sidebar_color: '#12' }))
    expect(screen.getByRole('button', { name: 'Save appearance' })).toBeDisabled()
  })

  it('puts the defaults back for review before saving', async () => {
    look.appearance = { ...saved, font: 'barlow', corners: 'sharp' }
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('button', { name: 'Back to defaults' }))

    expect(look.preview).toHaveBeenLastCalledWith(DEFAULT_APPEARANCE)
    expect(screen.getByRole('radio', { name: 'Atkinson Hyperlegible Next' })).toBeChecked()
  })

  it('shows the API error', async () => {
    look.save.mockRejectedValue(new CodedApiError('APPEARANCE_INVALID_FONT', 'bad font'))
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'Inter' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save appearance' }))

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('goes back to the saved look when leaving without saving', async () => {
    const { unmount } = render(<AppearanceSection />)
    await userEvent.click(screen.getByRole('radio', { name: 'Inter' }))

    unmount()

    expect(look.preview).toHaveBeenLastCalledWith(null)
  })

  it('is read-only for users who cannot change shop settings', () => {
    auth.role = 'customer_user'
    render(<AppearanceSection />)

    expect(screen.getByRole('radio', { name: 'Inter' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Save appearance' })).not.toBeInTheDocument()
  })

  it('explains that the classic design ignores it, with a way to switch', async () => {
    design.design = 'classic'
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('button', { name: 'Use the new design' }))

    expect(design.setDesign).toHaveBeenCalledWith('depot')
  })

  it('shows a load error', async () => {
    look.appearance = null
    look.error = 'Could not load'
    render(<AppearanceSection />)

    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Could not load'))
  })
})
