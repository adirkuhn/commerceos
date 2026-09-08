// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { AccountBlock } from '../AccountBlock'
import { AuthProvider } from '../../../auth/AuthContext'
import type { Account, AccountOrder } from '../../../types'

vi.mock('../../../api/auth', () => ({
  requestMagicLink: vi.fn(),
  verifySession: vi.fn(),
  getAccount: vi.fn(),
  listAccountOrders: vi.fn(),
  logout: vi.fn(),
}))

import * as authApi from '../../../api/auth'

const STORAGE_KEY = 'commerceos_session_token'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderBlock() {
  return render(
    <MemoryRouter initialEntries={['/account']}>
      <AuthProvider>
        <Routes>
          <Route path="/account" element={<AccountBlock />} />
          <Route path="/" element={<p>Home page</p>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('AccountBlock', () => {
  it('shows a sign-in prompt when not signed in', async () => {
    renderBlock()
    expect(await screen.findByText("You're not signed in.")).toBeInTheDocument()
  })

  it('shows the profile and order history when signed in', async () => {
    localStorage.setItem(STORAGE_KEY, 'session-1')
    const account: Account = { email: 'shopper@example.com', first_name: 'Ada', last_name: 'Lovelace' }
    const orders: AccountOrder[] = [
      { order_number: 'WEB-1', status: 'completed', fulfilled: true, total_minor_units: 1500, currency_code: 'EUR', placed_at: '2026-06-01T00:00:00Z' },
    ]
    vi.mocked(authApi.getAccount).mockResolvedValue(account)
    vi.mocked(authApi.listAccountOrders).mockResolvedValue(orders)

    renderBlock()

    expect(await screen.findByText('Ada Lovelace')).toBeInTheDocument()
    expect(screen.getByText('WEB-1')).toBeInTheDocument()
    expect(screen.getByText('€15.00')).toBeInTheDocument()
  })

  it('shows an empty state with no orders yet', async () => {
    localStorage.setItem(STORAGE_KEY, 'session-1')
    vi.mocked(authApi.getAccount).mockResolvedValue({ email: 'shopper@example.com', first_name: '', last_name: '' })
    vi.mocked(authApi.listAccountOrders).mockResolvedValue([])

    renderBlock()

    expect(await screen.findByText('No orders yet.')).toBeInTheDocument()
  })

  it('signs out and redirects to home', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'session-1')
    vi.mocked(authApi.getAccount).mockResolvedValue({ email: 'shopper@example.com', first_name: 'Ada', last_name: '' })
    vi.mocked(authApi.listAccountOrders).mockResolvedValue([])
    vi.mocked(authApi.logout).mockResolvedValue(null)

    renderBlock()
    await screen.findByText('Ada')

    await user.click(screen.getByRole('button', { name: 'Sign out' }))

    await waitFor(() => expect(screen.getByText('Home page')).toBeInTheDocument())
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })
})
