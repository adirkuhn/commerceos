// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, expect, it, beforeEach } from 'vitest'
import AccountCompanyPage from '../AccountCompanyPage'
import { renderWithProviders } from '@/test/testUtils'
import { useAuthStore } from '@/stores/authStore'
import { DEMO_CUSTOMER, DEMO_COMPANY_USERS } from '@/config/demoData'

function loginAsCompanyCustomer() {
  useAuthStore.setState({
    isAuthenticated: true,
    session: { customer: DEMO_CUSTOMER, token: 'test-token', expiresAt: '2099-01-01T00:00:00Z', twoFactorVerified: true },
  })
}

beforeEach(() => {
  useAuthStore.setState({ session: null, isAuthenticated: false })
})

describe('AccountCompanyPage', () => {
  it('shows the company name, price list, limits, and role for a B2B customer', () => {
    loginAsCompanyCustomer()
    renderWithProviders(<AccountCompanyPage />)

    expect(screen.getByRole('heading', { name: DEMO_CUSTOMER.company!.name })).toBeInTheDocument()
    expect(screen.getByText('Wholesale')).toBeInTheDocument()
    expect(screen.getByText('20% off every list price')).toBeInTheDocument()
    expect(screen.getByText('Your role: Owner')).toBeInTheDocument()
  })

  it('lists every company user with their role', () => {
    loginAsCompanyCustomer()
    renderWithProviders(<AccountCompanyPage />)

    for (const user of DEMO_COMPANY_USERS) {
      expect(screen.getByText(user.name)).toBeInTheDocument()
      expect(screen.getByText(user.email)).toBeInTheDocument()
    }
  })

  it('shows an empty state for a customer with no company', () => {
    useAuthStore.setState({
      isAuthenticated: true,
      session: { customer: { ...DEMO_CUSTOMER, company: undefined, companyRole: undefined }, token: 't', expiresAt: '2099-01-01T00:00:00Z', twoFactorVerified: true },
    })
    renderWithProviders(<AccountCompanyPage />)

    expect(screen.getByText('No company account')).toBeInTheDocument()
    expect(screen.queryByText('Company users')).not.toBeInTheDocument()
  })
})
