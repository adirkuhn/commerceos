import { describe, it, expect } from 'vitest'
import { customerName, emptyIdentity, identityComplete, identityFromCustomer, identityToInput } from '../customerIdentity'
import type { Customer } from '../../types'

const base: Customer = { id: 'c-1', tenant_id: 't', email: '', first_name: '', last_name: '', is_paying_customer: false, date_created: '', date_modified: '' }

describe('customerName', () => {
  it('uses the name the backend gives', () => {
    expect(customerName({ ...base, display_name: 'Claus B.V.', first_name: 'Kylian' })).toBe('Claus B.V.')
  })

  it('falls back to the person, the company, the email', () => {
    expect(customerName({ ...base, first_name: 'Ana', last_name: 'Souza' })).toBe('Ana Souza')
    expect(customerName({ ...base, customer_type: 'company', company_name: 'Sabor B.V.' })).toBe('Sabor B.V.')
    expect(customerName({ ...base, billing: { company: 'Shop' } })).toBe('Shop')
    expect(customerName({ ...base, email: 'a@x.nl' })).toBe('a@x.nl')
  })
})

describe('identityComplete', () => {
  it('a company needs its company name, nothing else', () => {
    expect(identityComplete({ ...emptyIdentity('company'), company_name: 'Sabor B.V.' })).toBe(true)
    expect(identityComplete({ ...emptyIdentity('company'), first_name: 'Ana', email: 'a@x.nl' })).toBe(false)
  })

  it('a private customer needs a name, email or phone', () => {
    expect(identityComplete(emptyIdentity('private'))).toBe(false)
    expect(identityComplete({ ...emptyIdentity('private'), last_name: 'Souza' })).toBe(true)
    expect(identityComplete({ ...emptyIdentity('private'), phone: '0612' })).toBe(true)
  })
})

describe('identityToInput', () => {
  it('sends a company with its contact person and no personal name', () => {
    const input = identityToInput({
      ...emptyIdentity('company'), company_name: ' Claus B.V. ', kvk_number: '34073293', vat_number: 'NL1B01', client_number: '3286',
      first_name: 'Kylian', last_name: 'Klockenbrink', email: 'info@claus.nl',
    }, { withContactPerson: true })

    expect(input).toEqual({
      customer_type: 'company', company_name: 'Claus B.V.', kvk_number: '34073293', vat_number: 'NL1B01', client_number: '3286',
      first_name: '', last_name: '', email: 'info@claus.nl',
      contact_persons: [{ first_name: 'Kylian', last_name: 'Klockenbrink' }],
    })
  })

  it('leaves the contact person out when none is named or wanted', () => {
    expect(identityToInput({ ...emptyIdentity('company'), company_name: 'Claus' }, { withContactPerson: true }).contact_persons).toBeUndefined()
    expect(identityToInput({ ...emptyIdentity('company'), company_name: 'Claus', first_name: 'K' }, { withContactPerson: false }).contact_persons).toBeUndefined()
  })

  it('sends a private customer by name, without company fields', () => {
    expect(identityToInput({ ...emptyIdentity('private'), first_name: 'Ana', company_name: 'old', kvk_number: '1' }, { withContactPerson: true }))
      .toEqual({ customer_type: 'private', company_name: '', kvk_number: '', vat_number: '', first_name: 'Ana', last_name: '', email: '' })
  })
})

describe('identityFromCustomer', () => {
  it('reads a company back, its person fields empty', () => {
    expect(identityFromCustomer({ ...base, customer_type: 'company', company_name: 'Claus', client_number: '7', kvk_number: '1', vat_number: 'NL', email: 'i@c.nl' }))
      .toEqual({ type: 'company', company_name: 'Claus', kvk_number: '1', vat_number: 'NL', client_number: '7', first_name: '', last_name: '', email: 'i@c.nl', phone: '' })
  })

  it('reads an older customer without a type as private', () => {
    expect(identityFromCustomer({ ...base, first_name: 'Ana', billing: { phone: '0612' } }).type).toBe('private')
    expect(identityFromCustomer({ ...base, first_name: 'Ana', billing: { phone: '0612' } }).phone).toBe('0612')
  })
})
