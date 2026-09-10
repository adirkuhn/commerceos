// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, beforeEach } from 'vitest'
import RegisterPage from '../RegisterPage'
import { renderWithProviders } from '@/test/testUtils'
import { useAuthStore } from '@/stores/authStore'
import { DEMO_CUSTOMER } from '@/config/demoData'

beforeEach(() => {
  useAuthStore.setState({ session: null, isAuthenticated: false })
})

describe('RegisterPage', () => {
  it('creates a new account and logs the user in', async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegisterPage />)

    await user.type(screen.getByLabelText('First name'), 'Jamie')
    await user.type(screen.getByLabelText('Last name'), 'Rivera')
    await user.type(screen.getByLabelText('Email'), `jamie-${Date.now()}@example.com`)
    await user.type(screen.getByLabelText('Password'), 'a-strong-password')
    await user.click(screen.getByRole('button', { name: 'Create account' }))

    await waitFor(() => expect(useAuthStore.getState().isAuthenticated).toBe(true))
    expect(useAuthStore.getState().session?.customer.firstName).toBe('Jamie')
  })

  it('rejects a password under 8 characters', async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegisterPage />)

    await user.type(screen.getByLabelText('First name'), 'Jamie')
    await user.type(screen.getByLabelText('Last name'), 'Rivera')
    await user.type(screen.getByLabelText('Email'), 'jamie@example.com')
    await user.type(screen.getByLabelText('Password'), 'short')
    await user.click(screen.getByRole('button', { name: 'Create account' }))

    expect(await screen.findByText('Password must be at least 8 characters')).toBeInTheDocument()
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('rejects registering an email that already has an account', async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegisterPage />)

    await user.type(screen.getByLabelText('First name'), 'Dup')
    await user.type(screen.getByLabelText('Last name'), 'Account')
    await user.type(screen.getByLabelText('Email'), DEMO_CUSTOMER.email)
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Create account' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('An account with this email already exists.')
  })
})
