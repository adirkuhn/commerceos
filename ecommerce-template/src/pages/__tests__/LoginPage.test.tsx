// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, beforeEach } from 'vitest'
import LoginPage from '../LoginPage'
import { renderWithProviders } from '@/test/testUtils'
import { useAuthStore } from '@/stores/authStore'
import { DEMO_CUSTOMER, DEMO_PASSWORD } from '@/config/demoData'

beforeEach(() => {
  useAuthStore.setState({ session: null, isAuthenticated: false })
})

describe('LoginPage', () => {
  it('logs in with the seeded demo account against the real mock auth flow', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    await user.type(screen.getByLabelText('Email'), DEMO_CUSTOMER.email)
    await user.type(screen.getByLabelText('Password'), DEMO_PASSWORD)
    await user.click(screen.getByRole('button', { name: 'Log in' }))

    await waitFor(() => expect(useAuthStore.getState().isAuthenticated).toBe(true))
    expect(useAuthStore.getState().session?.customer.email).toBe(DEMO_CUSTOMER.email)
  })

  it('shows an error for the wrong password and never authenticates', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    await user.type(screen.getByLabelText('Email'), DEMO_CUSTOMER.email)
    await user.type(screen.getByLabelText('Password'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Incorrect email or password.')
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('validates a malformed email client-side before submitting', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    await user.type(screen.getByLabelText('Email'), 'not-an-email')
    await user.type(screen.getByLabelText('Password'), 'whatever')
    await user.click(screen.getByRole('button', { name: 'Log in' }))

    expect(await screen.findByText('Enter a valid email address')).toBeInTheDocument()
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })
})
