// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CompanyProfileSection } from '../CompanyProfileSection'
import { CompanyProfileError } from '../api'
import type { CompanyProfile } from '../types'

const auth = vi.hoisted(() => ({ role: 'customer_owner' }))
vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: auth.role }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../api', async importOriginal => {
  const actual = await importOriginal<typeof import('../api')>()
  return { ...actual, getCompanyProfile: vi.fn(), updateCompanyProfile: vi.fn() }
})

import * as api from '../api'

const saved: CompanyProfile = {
  legal_name: 'Cantinho B.V.', trading_name: 'Cantinho', vat_number: 'NL123456789B01', chamber_of_commerce: '12345678',
  email: 'info@example.nl', phone: '+31 20 123 4567', iban: 'NL91ABNA0417164300',
  address: { street: 'Damrak', house_number: '1', postal_code: '1012 LG', city: 'Amsterdam', country: 'NL' },
  updated_at: '2026-09-24T10:00:00Z',
}

beforeEach(() => {
  vi.clearAllMocks()
  auth.role = 'customer_owner'
  vi.mocked(api.getCompanyProfile).mockResolvedValue(saved)
})

describe('CompanyProfileSection', () => {
  it('loads the saved profile into labelled fields', async () => {
    render(<CompanyProfileSection />)

    expect(await screen.findByLabelText('Legal name')).toHaveValue('Cantinho B.V.')
    expect(screen.getByLabelText('VAT number')).toHaveValue('NL123456789B01')
    expect(screen.getByLabelText('City')).toHaveValue('Amsterdam')
    expect(screen.getByLabelText('Country')).toHaveValue('NL')
  })

  it('tells the browser what to autofill', async () => {
    render(<CompanyProfileSection />)
    expect(await screen.findByLabelText('Legal name')).toHaveAttribute('autocomplete', 'organization')
    expect(screen.getByLabelText('Email')).toHaveAttribute('autocomplete', 'email')
    expect(screen.getByLabelText('Postal code')).toHaveAttribute('autocomplete', 'postal-code')
  })

  it('only enables Save once something changed, then saves the whole profile and confirms', async () => {
    vi.mocked(api.updateCompanyProfile).mockResolvedValue({ ...saved, legal_name: 'Cantinho Brasileiro B.V.' })
    render(<CompanyProfileSection />)
    const name = await screen.findByLabelText('Legal name')
    const save = screen.getByRole('button', { name: 'Save changes' })
    expect(save).toBeDisabled()

    await userEvent.clear(name)
    await userEvent.type(name, 'Cantinho Brasileiro B.V.')
    expect(save).toBeEnabled()
    await userEvent.click(save)

    await waitFor(() => expect(api.updateCompanyProfile).toHaveBeenCalledWith('test-token', expect.objectContaining({
      legal_name: 'Cantinho Brasileiro B.V.', vat_number: 'NL123456789B01',
      address: expect.objectContaining({ city: 'Amsterdam', country: 'NL' }),
    })))
    expect(await screen.findByRole('status')).toHaveTextContent('Saved')
    expect(save).toBeDisabled()
  })

  it('shows a validation error next to its field and moves focus there', async () => {
    vi.mocked(api.updateCompanyProfile).mockRejectedValue(new CompanyProfileError('INVALID_IBAN', 'IBAN is not valid'))
    render(<CompanyProfileSection />)
    const iban = await screen.findByLabelText('IBAN')
    await userEvent.clear(iban)
    await userEvent.type(iban, 'NL00BAD')

    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(iban).toHaveAttribute('aria-invalid', 'true'))
    expect(iban).toHaveAccessibleDescription('IBAN is not valid')
    expect(iban).toHaveFocus()
  })

  it('shows errors that belong to no field above the form', async () => {
    vi.mocked(api.updateCompanyProfile).mockRejectedValue(new CompanyProfileError('INTERNAL', 'internal server error'))
    render(<CompanyProfileSection />)
    await userEvent.type(await screen.findByLabelText('Phone'), '9')

    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('internal server error')
  })

  it('is read-only for users who cannot change settings', async () => {
    auth.role = 'customer_user'
    render(<CompanyProfileSection />)

    expect(await screen.findByLabelText('Legal name')).toHaveAttribute('readonly')
    expect(screen.queryByRole('button', { name: 'Save changes' })).not.toBeInTheDocument()
    expect(screen.getByText(/Only owners and admins can change/)).toBeInTheDocument()
  })

  it('reports a load failure', async () => {
    vi.mocked(api.getCompanyProfile).mockRejectedValue(new CompanyProfileError('INTERNAL', 'internal server error'))
    render(<CompanyProfileSection />)
    expect(await screen.findByRole('alert')).toHaveTextContent('internal server error')
  })
})
