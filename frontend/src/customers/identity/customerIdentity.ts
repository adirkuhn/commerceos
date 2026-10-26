import { fullName } from '../../format/personName'
import type { Customer, CustomerInput, CustomerType } from '../types'

// What names a customer in the customer form and the order's new-customer
// dialog. For a company, first/last name are its (first) contact person.
export interface IdentityDraft {
  type: CustomerType
  company_name: string
  kvk_number: string
  vat_number: string
  client_number: string
  first_name: string
  last_name: string
  email: string
  phone: string
}

export function emptyIdentity(type: CustomerType): IdentityDraft {
  return { type, company_name: '', kvk_number: '', vat_number: '', client_number: '', first_name: '', last_name: '', email: '', phone: '' }
}

// customerName is how a customer is named everywhere: the backend's
// display_name, or the same rule when an older response lacks it.
export function customerName(c: Customer): string {
  if (c.display_name) return c.display_name
  if (c.customer_type === 'company' && c.company_name) return c.company_name
  return [c.first_name, c.last_name].filter(Boolean).join(' ') || c.billing?.company || fullName(c)
}

export function identityFromCustomer(c: Customer): IdentityDraft {
  return {
    type: c.customer_type ?? 'private', company_name: c.company_name ?? '', kvk_number: c.kvk_number ?? '',
    vat_number: c.vat_number ?? '', client_number: c.client_number ?? '',
    first_name: c.first_name ?? '', last_name: c.last_name ?? '', email: c.email ?? '', phone: c.billing?.phone ?? '',
  }
}

// identityComplete mirrors the backend's rule: a company needs its company
// name; a private customer a name, email or phone. Email is never required.
export function identityComplete(d: IdentityDraft): boolean {
  if (d.type === 'company') return d.company_name.trim() !== ''
  return [d.first_name, d.last_name, d.email, d.phone].some(v => v.trim() !== '')
}

// identityToInput is the customer fields to save. A company keeps no
// personal name -- the person is sent as its first contact person when
// withContactPerson (on create). Phone goes in the address, by the caller.
// An empty client number is left out: the backend numbers a new customer
// and keeps an existing one's.
export function identityToInput(d: IdentityDraft, opts: { withContactPerson: boolean }): CustomerInput {
  const clientNumber = d.client_number.trim() ? { client_number: d.client_number.trim() } : {}
  if (d.type === 'private') {
    return {
      customer_type: 'private', company_name: '', kvk_number: '', vat_number: '', ...clientNumber,
      first_name: d.first_name.trim(), last_name: d.last_name.trim(), email: d.email.trim(),
    }
  }
  const input: CustomerInput = {
    customer_type: 'company', company_name: d.company_name.trim(), kvk_number: d.kvk_number.trim(), vat_number: d.vat_number.trim(),
    ...clientNumber, first_name: '', last_name: '', email: d.email.trim(),
  }
  const person = { first_name: d.first_name.trim(), last_name: d.last_name.trim() }
  if (opts.withContactPerson && (person.first_name || person.last_name)) input.contact_persons = [person]
  return input
}
