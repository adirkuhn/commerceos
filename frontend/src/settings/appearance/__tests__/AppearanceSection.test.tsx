// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
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

import { AppearanceSection } from '../AppearanceSection'

const saved: Appearance = { ...DEFAULT_APPEARANCE, updated_at: '2026-10-12T10:00:00Z' }

beforeEach(() => {
  vi.clearAllMocks()
  auth.role = 'customer_owner'
  look.appearance = saved
  look.error = ''
  look.save.mockImplementation(async (a: typeof DEFAULT_APPEARANCE) => ({ ...a, updated_at: '2026-10-12T11:00:00Z' }))
})

describe('AppearanceSection', () => {
  it('shows the saved appearance', () => {
    render(<AppearanceSection />)

    const group = screen.getByRole('radiogroup', { name: 'Default look' })
    expect(within(group).getByRole('radio', { name: 'Ticket Wallet' })).toBeChecked()
    expect(within(group).getByRole('radio', { name: 'Graphite' })).not.toBeChecked()
    expect(screen.getByRole('radio', { name: 'Comfortable' })).toBeChecked()
  })

  it('describes each look and says people can still pick their own', () => {
    render(<AppearanceSection />)

    expect(screen.getByRole('radio', { name: 'Graphite' })).toHaveAccessibleDescription('Graphite panels with blue accents')
    expect(screen.getByRole('radio', { name: 'Ticket Wallet' })).toHaveAccessibleDescription('Navy and coupon white, purple figures')
    expect(screen.getByText(/unless they choose their own in the account menu/)).toBeInTheDocument()
  })

  it('previews each change on the whole app before saving', async () => {
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'Graphite' }))
    expect(look.preview).toHaveBeenLastCalledWith({ ...DEFAULT_APPEARANCE, default_look: 'graphite' })
    expect(screen.getByRole('radio', { name: 'Graphite' })).toBeChecked()

    await userEvent.click(screen.getByRole('radio', { name: 'Round' }))
    expect(look.preview).toHaveBeenLastCalledWith({ ...DEFAULT_APPEARANCE, default_look: 'graphite', corners: 'round' })
    expect(look.save).not.toHaveBeenCalled()
  })

  it('saves the new appearance', async () => {
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'Graphite' }))
    await userEvent.click(screen.getByRole('radio', { name: 'Larger' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save appearance' }))

    expect(look.save).toHaveBeenCalledWith({ default_look: 'graphite', corners: 'normal', text_size: 'larger', density: 'comfortable' })
    expect(await screen.findByRole('status')).toHaveTextContent('Saved')
  })

  it('sends only the appearance fields, not updated_at', async () => {
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'Compact' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save appearance' }))

    expect(look.save).toHaveBeenCalledWith({ ...DEFAULT_APPEARANCE, density: 'compact' })
  })

  it('has nothing to save until something changes', () => {
    render(<AppearanceSection />)

    expect(screen.getByRole('button', { name: 'Save appearance' })).toBeDisabled()
  })

  it('puts the defaults back for review before saving', async () => {
    look.appearance = { ...saved, default_look: 'graphite', corners: 'sharp' }
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('button', { name: 'Back to defaults' }))

    expect(look.preview).toHaveBeenLastCalledWith(DEFAULT_APPEARANCE)
    expect(screen.getByRole('radio', { name: 'Ticket Wallet' })).toBeChecked()
  })

  it('shows the API error', async () => {
    look.save.mockRejectedValue(new CodedApiError('APPEARANCE_INVALID_LOOK', 'bad look'))
    render(<AppearanceSection />)

    await userEvent.click(screen.getByRole('radio', { name: 'Graphite' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save appearance' }))

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('goes back to the saved look when leaving without saving', async () => {
    const { unmount } = render(<AppearanceSection />)
    await userEvent.click(screen.getByRole('radio', { name: 'Graphite' }))

    unmount()

    expect(look.preview).toHaveBeenLastCalledWith(null)
  })

  it('is read-only for users who cannot change shop settings', () => {
    auth.role = 'customer_user'
    render(<AppearanceSection />)

    expect(screen.getByRole('radio', { name: 'Graphite' })).toBeDisabled()
    expect(screen.getByRole('radio', { name: 'Round' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Save appearance' })).not.toBeInTheDocument()
  })

  it('shows a load error', async () => {
    look.appearance = null
    look.error = 'Could not load'
    render(<AppearanceSection />)

    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Could not load'))
  })
})
