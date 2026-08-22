// @vitest-environment jsdom
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { AcceptInvite } from '../AcceptInvite'

vi.mock('../../team/api', () => ({
  acceptInvite: vi.fn(),
}))

import * as teamApi from '../../team/api'

function renderWithToken(token: string | null) {
  const url = token ? `/accept-invite?token=${token}` : '/accept-invite'
  window.history.pushState({}, '', url)
  return render(<AcceptInvite />)
}

beforeEach(() => { vi.clearAllMocks() })

describe('AcceptInvite', () => {
  it('shows an error when no token is in the URL', () => {
    renderWithToken(null)
    expect(screen.getByText(/no invite token/i)).toBeInTheDocument()
  })

  it('accepts the invite for a new account (name + password)', async () => {
    vi.mocked(teamApi.acceptInvite).mockResolvedValue(null)
    renderWithToken('raw-token-123')

    fireEvent.change(screen.getByLabelText(/first name/i), { target: { value: 'Bo' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'secret123' } })
    fireEvent.click(screen.getByRole('button', { name: /accept invite/i }))

    await waitFor(() => expect(teamApi.acceptInvite).toHaveBeenCalledWith({
      token: 'raw-token-123', firstName: 'Bo', lastName: '', password: 'secret123',
    }))
    expect(await screen.findByText(/you're in/i)).toBeInTheDocument()
  })

  it('accepts the invite for an already-existing lm2 account with no fields filled', async () => {
    vi.mocked(teamApi.acceptInvite).mockResolvedValue(null)
    renderWithToken('raw-token-123')

    fireEvent.click(screen.getByRole('button', { name: /accept invite/i }))

    await waitFor(() => expect(teamApi.acceptInvite).toHaveBeenCalledWith({
      token: 'raw-token-123', firstName: '', lastName: '', password: '',
    }))
  })

  it('shows an error from an invalid or expired invite', async () => {
    vi.mocked(teamApi.acceptInvite).mockRejectedValue(new Error('invite is invalid, expired, or already used'))
    renderWithToken('bad-token')

    fireEvent.click(screen.getByRole('button', { name: /accept invite/i }))

    expect(await screen.findByText('invite is invalid, expired, or already used')).toBeInTheDocument()
  })
})
