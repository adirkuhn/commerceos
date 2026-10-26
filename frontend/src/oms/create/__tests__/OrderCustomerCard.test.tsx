// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../../customers/api', () => ({ getCustomer: vi.fn(), createCustomer: vi.fn(), listContactPersons: vi.fn() }))
vi.mock('../../../search/api', () => ({ globalSearch: vi.fn() }))
import { listContactPersons } from '../../../customers/api'
import { OrderCustomerCard } from '../OrderCustomerCard'
import type { ContactPerson, Customer } from '../../../customers/types'

const claus: Customer = {
  id: 'c-1', tenant_id: 't', email: '', first_name: '', last_name: '', is_paying_customer: false, date_created: '', date_modified: '',
  customer_type: 'company', company_name: 'Claus B.V.', display_name: 'Claus B.V.',
}
const person = (id: string, first: string, email = ''): ContactPerson => ({ id, customer_id: 'c-1', first_name: first, last_name: 'K', role: '', email, phone: '', created_at: '' })

beforeEach(() => vi.clearAllMocks())

describe('OrderCustomerCard contact person', () => {
  it('picks the company\'s first contact person, and another when chosen', async () => {
    vi.mocked(listContactPersons).mockResolvedValue([person('cp-1', 'Kylian'), person('cp-2', 'Sara', 'sara@claus.nl')])
    const onContactPerson = vi.fn()
    render(<OrderCustomerCard token="tok" customer={claus} onChange={vi.fn()} onContactPerson={onContactPerson} />)

    const select = await screen.findByLabelText('Contact person')
    await waitFor(() => expect(onContactPerson).toHaveBeenCalledWith(expect.objectContaining({ id: 'cp-1' })))
    expect(select).toHaveValue('cp-1')

    fireEvent.change(select, { target: { value: 'cp-2' } })
    expect(onContactPerson).toHaveBeenLastCalledWith(expect.objectContaining({ id: 'cp-2', email: 'sara@claus.nl' }))
  })

  it('shows no contact person choice for a private customer', () => {
    render(<OrderCustomerCard token="tok" customer={{ ...claus, customer_type: 'private', first_name: 'Ana' }} onChange={vi.fn()} onContactPerson={vi.fn()} />)
    expect(screen.queryByLabelText('Contact person')).not.toBeInTheDocument()
    expect(listContactPersons).not.toHaveBeenCalled()
  })

  it('shows no choice for a company without contact persons', async () => {
    vi.mocked(listContactPersons).mockResolvedValue([])
    render(<OrderCustomerCard token="tok" customer={claus} onChange={vi.fn()} onContactPerson={vi.fn()} />)
    await waitFor(() => expect(listContactPersons).toHaveBeenCalled())
    expect(screen.queryByLabelText('Contact person')).not.toBeInTheDocument()
  })
})

describe('OrderCustomerCard on an order being edited', () => {
  it('shows the order\'s own contact person and leaves the billing name alone', async () => {
    vi.mocked(listContactPersons).mockResolvedValue([person('cp-1', 'Kylian'), person('cp-2', 'Sara')])
    const onContactPerson = vi.fn()
    render(<OrderCustomerCard token="tok" customer={claus} onChange={vi.fn()} onContactPerson={onContactPerson} current={{ first_name: 'Sara', last_name: 'K' }} />)

    expect(await screen.findByLabelText('Contact person')).toHaveValue('cp-2')
    expect(onContactPerson).not.toHaveBeenCalled()
  })

  it('selects no one when the order is addressed to someone else', async () => {
    vi.mocked(listContactPersons).mockResolvedValue([person('cp-1', 'Kylian')])
    const onContactPerson = vi.fn()
    render(<OrderCustomerCard token="tok" customer={claus} onChange={vi.fn()} onContactPerson={onContactPerson} current={{ first_name: 'Bruno', last_name: 'L' }} />)

    expect(await screen.findByLabelText('Contact person')).toHaveValue('')
    expect(onContactPerson).not.toHaveBeenCalled()
  })
})

describe('OrderCustomerCard on an edited order with another customer chosen', () => {
  it('starts the new company on its first contact person', async () => {
    vi.mocked(listContactPersons).mockResolvedValue([person('cp-1', 'Kylian')])
    const onContactPerson = vi.fn()
    const { rerender } = render(<OrderCustomerCard token="tok" customer={null} onChange={vi.fn()} onContactPerson={onContactPerson} current={{ first_name: 'Bruno', last_name: 'L' }} />)
    rerender(<OrderCustomerCard token="tok" customer={claus} onChange={vi.fn()} onContactPerson={onContactPerson} current={{ first_name: 'Bruno', last_name: 'L' }} />)

    await waitFor(() => expect(onContactPerson).toHaveBeenCalledWith(expect.objectContaining({ id: 'cp-1' })))
  })
})
