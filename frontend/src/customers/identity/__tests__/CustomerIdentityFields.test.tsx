// @vitest-environment jsdom
import { useState } from 'react'
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CustomerIdentityFields } from '../CustomerIdentityFields'
import { emptyIdentity, type IdentityDraft } from '../customerIdentity'

let latest: IdentityDraft = emptyIdentity('company')

function Harness({ initial, contactPerson = true, phone = false }: { initial: IdentityDraft; contactPerson?: boolean; phone?: boolean }) {
  const [value, setValue] = useState(initial)
  latest = value
  return <CustomerIdentityFields idPrefix="t" value={value} onChange={setValue} showContactPerson={contactPerson} showPhone={phone} />
}

describe('CustomerIdentityFields', () => {
  it('a company asks for company name, KVK, VAT ID and a contact person', () => {
    render(<Harness initial={emptyIdentity('company')} />)

    expect(screen.getByRole('radio', { name: 'Company' })).toHaveAttribute('aria-checked', 'true')
    fireEvent.change(screen.getByLabelText('Company name'), { target: { value: 'Claus B.V.' } })
    fireEvent.change(screen.getByLabelText('KVK number'), { target: { value: '34073293' } })
    fireEvent.change(screen.getByLabelText('VAT ID'), { target: { value: 'NL1B01' } })
    expect(screen.getByRole('group', { name: 'Contact person' })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Contact first name'), { target: { value: 'Kylian' } })

    expect(latest).toMatchObject({ type: 'company', company_name: 'Claus B.V.', kvk_number: '34073293', vat_number: 'NL1B01', first_name: 'Kylian' })
  })

  it('switching to private asks for the person instead', () => {
    render(<Harness initial={emptyIdentity('company')} />)

    fireEvent.click(screen.getByRole('radio', { name: 'Private' }))

    expect(latest.type).toBe('private')
    expect(screen.queryByLabelText('Company name')).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Ana' } })
    expect(latest.first_name).toBe('Ana')
  })

  it('email is optional and the client number is filled in automatically when left empty', () => {
    render(<Harness initial={emptyIdentity('private')} phone />)

    expect(screen.getByLabelText('Email (optional)')).not.toBeRequired()
    expect(screen.getByLabelText('Client number')).toHaveAttribute('placeholder', 'Automatic')
    fireEvent.change(screen.getByLabelText('Phone'), { target: { value: '0612' } })
    expect(latest.phone).toBe('0612')
  })

  it('hides the contact person when editing a company', () => {
    render(<Harness initial={emptyIdentity('company')} contactPerson={false} />)
    expect(screen.queryByRole('group', { name: 'Contact person' })).not.toBeInTheDocument()
  })

  it('the type buttons do not submit a surrounding form', () => {
    render(<Harness initial={emptyIdentity('company')} />)
    expect(screen.getByRole('radio', { name: 'Private' })).toHaveAttribute('type', 'button')
  })
})
