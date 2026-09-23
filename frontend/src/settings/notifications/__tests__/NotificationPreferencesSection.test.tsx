// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { NotificationPreferencesSection } from '../NotificationPreferencesSection'
import { NotificationPreferencesError } from '../api'
import type { NotificationPreferences } from '../types'

const auth = vi.hoisted(() => ({ role: 'customer_owner' }))
vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: auth.role } }),
}))
vi.mock('../api', async importOriginal => {
  const actual = await importOriginal<typeof import('../api')>()
  return { ...actual, getNotificationPreferences: vi.fn(), updateNotificationPreferences: vi.fn() }
})
import * as api from '../api'

const defaults: NotificationPreferences = {
  categories: [
    { category: 'imports', in_app: true, email: false },
    { category: 'purchase_documents', in_app: true, email: false },
    { category: 'channel_offers', in_app: true, email: false },
    { category: 'other', in_app: true, email: false },
  ],
  email_recipients: [],
  updated_at: null,
}

beforeEach(() => {
  vi.clearAllMocks()
  auth.role = 'customer_owner'
  vi.mocked(api.getNotificationPreferences).mockResolvedValue(defaults)
})

describe('NotificationPreferencesSection', () => {
  it('shows each category with a labelled bell and email choice', async () => {
    render(<NotificationPreferencesSection />)

    expect(await screen.findByRole('checkbox', { name: 'Show imports in the bell' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Email imports' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Email purchase documents' })).toBeInTheDocument()
    expect(screen.getByText('Catalog, customer and order imports from your channels')).toBeInTheDocument()
  })

  it('saves the chosen channels and recipients', async () => {
    vi.mocked(api.updateNotificationPreferences).mockImplementation(async (_t, input) => ({ ...input, updated_at: '2026-09-24T10:00:00Z' }))
    render(<NotificationPreferencesSection />)

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Email imports' }))
    await userEvent.type(screen.getByLabelText('Email recipients'), 'ops@example.nl{enter}boss@example.nl')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(api.updateNotificationPreferences).toHaveBeenCalledWith('test-token', {
      categories: expect.arrayContaining([{ category: 'imports', in_app: true, email: true }]),
      email_recipients: ['ops@example.nl', 'boss@example.nl'],
    }))
    expect(await screen.findByRole('status')).toHaveTextContent('Saved')
  })

  it('puts a recipient error on the recipients field', async () => {
    vi.mocked(api.updateNotificationPreferences).mockRejectedValue(
      new NotificationPreferencesError('EMAIL_RECIPIENTS_REQUIRED', 'add at least one email recipient to send notifications by email'),
    )
    render(<NotificationPreferencesSection />)

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Email imports' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    const field = screen.getByLabelText('Email recipients')
    await waitFor(() => expect(field).toHaveAttribute('aria-invalid', 'true'))
    expect(field).toHaveAccessibleDescription(expect.stringContaining('add at least one email recipient'))
    expect(field).toHaveFocus()
  })

  it('is read-only for users who cannot change settings', async () => {
    auth.role = 'customer_user'
    render(<NotificationPreferencesSection />)

    expect(await screen.findByRole('checkbox', { name: 'Email imports' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Save changes' })).not.toBeInTheDocument()
  })

  it('reports a load failure', async () => {
    vi.mocked(api.getNotificationPreferences).mockRejectedValue(new NotificationPreferencesError('INTERNAL', 'internal server error'))
    render(<NotificationPreferencesSection />)
    expect(await screen.findByRole('alert')).toHaveTextContent('internal server error')
  })
})
