// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { LoginFormBlock } from '../LoginFormBlock'
import { AuthProvider } from '../../../auth/AuthContext'
import type { Session } from '../../../types'

vi.mock('../../../api/auth', () => ({
  requestMagicLink: vi.fn(),
  verifySession: vi.fn(),
  getAccount: vi.fn(),
  listAccountOrders: vi.fn(),
  logout: vi.fn(),
}))

import * as authApi from '../../../api/auth'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginFormBlock />} />
          <Route path="/account" element={<p>Account page</p>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('LoginFormBlock', () => {
  it('sends a magic link and shows the check-your-email state', async () => {
    const user = userEvent.setup()
    vi.mocked(authApi.requestMagicLink).mockResolvedValue(null)

    renderAt('/login')
    await user.type(screen.getByLabelText('Email'), 'shopper@example.com')
    await user.click(screen.getByRole('button', { name: 'Send sign-in link' }))

    expect(authApi.requestMagicLink).toHaveBeenCalledWith('shopper@example.com', expect.stringContaining('/login'))
    expect(await screen.findByText('Check your email')).toBeInTheDocument()
  })

  it('shows an error when the request fails', async () => {
    const user = userEvent.setup()
    vi.mocked(authApi.requestMagicLink).mockRejectedValue(new Error('server error'))

    renderAt('/login')
    await user.type(screen.getByLabelText('Email'), 'shopper@example.com')
    await user.click(screen.getByRole('button', { name: 'Send sign-in link' }))

    expect(await screen.findByText('server error')).toBeInTheDocument()
  })

  it('completes sign-in automatically when a token is in the URL and redirects to account', async () => {
    const session: Session = { session_token: 'session-1', account: { email: 'shopper@example.com', first_name: 'Ada', last_name: '' } }
    vi.mocked(authApi.verifySession).mockResolvedValue(session)

    renderAt('/login?token=LINK123')

    expect(await screen.findByText('Account page')).toBeInTheDocument()
    expect(authApi.verifySession).toHaveBeenCalledWith('LINK123')
  })

  it('shows an error for an invalid/expired token instead of redirecting', async () => {
    vi.mocked(authApi.verifySession).mockRejectedValue(new Error('this sign-in link is invalid or has expired'))

    renderAt('/login?token=bad-token')

    expect(await screen.findByText('this sign-in link is invalid or has expired')).toBeInTheDocument()
    expect(screen.queryByText('Account page')).not.toBeInTheDocument()
  })
})
