// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CompanyDetailsCard } from '../CompanyDetailsCard'
import type { Customer } from '../../types'

const company: Customer = {
  id: 'c-1', tenant_id: 't', email: '', first_name: '', last_name: '', is_paying_customer: false, date_created: '', date_modified: '',
  customer_type: 'company', company_name: 'Claus Hoofddorp B.V.', client_number: '3286', kvk_number: '34073293', vat_number: 'NL001B01',
  billing: { address_1: 'Bosweg 19', postcode: '2131LX', city: 'Hoofddorp', country: 'NL' },
}

describe('CompanyDetailsCard', () => {
  it('shows client number, KVK number linking to the KVK and VAT ID', () => {
    render(<CompanyDetailsCard customer={company} />)

    expect(screen.getByText('3286')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /34073293/ })).toHaveAttribute('href', 'https://www.kvk.nl/bestellen/#/34073293')
    expect(screen.getByText('NL001B01')).toBeInTheDocument()
  })

  it('links the address to Google Maps', () => {
    render(<CompanyDetailsCard customer={company} />)
    expect(screen.getByRole('link', { name: 'Open address in Google Maps' }))
      .toHaveAttribute('href', 'https://www.google.com/maps/search/?api=1&query=Bosweg%2019%2C%202131LX%20Hoofddorp%2C%20NL')
  })

  it('leaves out what is not filled in', () => {
    render(<CompanyDetailsCard customer={{ ...company, kvk_number: '', vat_number: '', billing: {} }} />)
    expect(screen.queryByText('KVK number')).not.toBeInTheDocument()
    expect(screen.queryByText('VAT ID')).not.toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
