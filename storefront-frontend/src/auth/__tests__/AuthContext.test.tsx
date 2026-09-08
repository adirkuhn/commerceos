// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthProvider, useAuth } from '../AuthContext'
import type { Session, Account } from '../../types'

vi.mock('../../api/auth', () => ({
  requestMagicLink: vi.fn(),
  verifySession: vi.fn(),
  getAccount: vi.fn(),
  listAccountOrders: vi.fn(),
  logout: vi.fn(),
}))

import * as authApi from '../../api/auth'

const STORAGE_KEY = 'commerceos_session_token'

function Harness() {
  const { account, loading, requestLogin, completeLogin, logout, error } = useAuth()
  return (
    <div>
      <p data-testid="account">{loading ? 'loading' : account ? account.email : 'signed out'}</p>
      {error && <p data-testid="error">{error}</p>}
      <button onClick={() => { requestLogin('shopper@example.com', 'https://x/login').catch(() => {}) }}>Request link</button>
      <button onClick={() => { completeLogin('LINK123').catch(() => {}) }}>Complete login</button>
      <button onClick={() => logout()}>Sign out</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

describe('AuthContext', () => {
  it('starts signed out with no stored session', async () => {
    render(<AuthProvider><Harness /></AuthProvider>)
    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('signed out'))
    expect(authApi.getAccount).not.toHaveBeenCalled()
  })

  it('loads the account on mount when a session token is already stored', async () => {
    localStorage.setItem(STORAGE_KEY, 'session-9')
    const account: Account = { email: 'shopper@example.com', first_name: 'Ada', last_name: '' }
    vi.mocked(authApi.getAccount).mockResolvedValue(account)

    render(<AuthProvider><Harness /></AuthProvider>)

    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('shopper@example.com'))
    expect(authApi.getAccount).toHaveBeenCalledWith('session-9')
  })

  it('requestLogin calls the API with the given email/redirect', async () => {
    const user = userEvent.setup()
    vi.mocked(authApi.requestMagicLink).mockResolvedValue(null)

    render(<AuthProvider><Harness /></AuthProvider>)
    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('signed out'))

    await user.click(screen.getByText('Request link'))

    await waitFor(() => expect(authApi.requestMagicLink).toHaveBeenCalledWith('shopper@example.com', 'https://x/login'))
  })

  it('completeLogin stores the session and sets the account', async () => {
    const user = userEvent.setup()
    const session: Session = { session_token: 'session-1', account: { email: 'shopper@example.com', first_name: 'Ada', last_name: '' } }
    vi.mocked(authApi.verifySession).mockResolvedValue(session)

    render(<AuthProvider><Harness /></AuthProvider>)
    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('signed out'))

    await user.click(screen.getByText('Complete login'))

    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('shopper@example.com'))
    expect(localStorage.getItem(STORAGE_KEY)).toBe('session-1')
  })

  it('logout clears the stored session', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'session-9')
    vi.mocked(authApi.getAccount).mockResolvedValue({ email: 'shopper@example.com', first_name: '', last_name: '' })
    vi.mocked(authApi.logout).mockResolvedValue(null)

    render(<AuthProvider><Harness /></AuthProvider>)
    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('shopper@example.com'))

    await user.click(screen.getByText('Sign out'))

    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('signed out'))
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })

  it('surfaces an error from a rejected completeLogin', async () => {
    const user = userEvent.setup()
    vi.mocked(authApi.verifySession).mockRejectedValue(new Error('this sign-in link is invalid or has expired'))

    render(<AuthProvider><Harness /></AuthProvider>)
    await waitFor(() => expect(screen.getByTestId('account')).toHaveTextContent('signed out'))

    await user.click(screen.getByText('Complete login'))

    await waitFor(() => expect(screen.getByTestId('error')).toHaveTextContent('this sign-in link is invalid or has expired'))
  })
})
