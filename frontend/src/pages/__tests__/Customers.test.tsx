// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Customers from '../Customers'
import { renderAt } from '../../test/renderAt'
import { act } from 'react'
import { MemoryRouter, useNavigate } from 'react-router-dom'
import type { Customer } from '../../customers/types'
import type { SalesOrder } from '../../oms/types'

// Stands in for CommandPalette's openRecord: a `navigate()` call to a
// per-record URL fired while Customers is already mounted, without
// Customers itself remounting.
function NavigateToCustomerButton({ id }: { id: string }) {
  const navigate = useNavigate()
  return <button onClick={() => navigate(`/customers/${id}`)}>simulate search result click</button>
}

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../customers/api', () => ({
  listCustomers: vi.fn(),
  createCustomer: vi.fn(),
  updateCustomer: vi.fn(),
  deleteCustomer: vi.fn(),
  listCustomerWhatsAppMessages: vi.fn(),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
}))

import * as customersApi from '../../customers/api'
import * as omsApi from '../../oms/api'
import { acceptConfirm, dismissConfirm } from '../../test/confirm'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'pos', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'completed', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    ...overrides,
  }
}

function customer(overrides: Partial<Customer> = {}): Customer {
  return {
    id: 'customer-1', tenant_id: 'tenant-1', email: 'jane@example.com',
    first_name: 'Jane', last_name: 'Doe', is_paying_customer: false,
    date_created: '', date_modified: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(customersApi.listCustomers).mockResolvedValue([])
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })
  vi.mocked(customersApi.listCustomerWhatsAppMessages).mockResolvedValue([])
})

describe('Customers list', () => {
  it('shows an empty state when there are no customers', async () => {
    render(<MemoryRouter><Customers /></MemoryRouter>)

    expect(await screen.findByText('No customers yet')).toBeInTheDocument()
  })

  it('lists customers', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('jane@example.com')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([
      customer(),
      customer({ id: 'customer-2', first_name: 'Bob', last_name: 'Smith', email: 'bob@example.com' }),
    ])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await screen.findByText('Jane Doe')

    await userEvent.type(screen.getByPlaceholderText('Search by name or email...'), 'Bob')

    expect(screen.queryByText('Jane Doe')).not.toBeInTheDocument()
    expect(screen.getByText('Bob Smith')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(customersApi.deleteCustomer).toHaveBeenCalledWith('test-token', 'customer-1'))
    expect(screen.queryByText('Add Customer')).toBeInTheDocument()
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    await dismissConfirm()
    expect(customersApi.deleteCustomer).not.toHaveBeenCalled()
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
  })

  it('shows paying customer status in the row', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer({ is_paying_customer: true })])

    render(<MemoryRouter><Customers /></MemoryRouter>)

    expect(await screen.findByText('Paying')).toBeInTheDocument()
  })

  it('paginates when there are more than 20 customers', async () => {
    const many = Array.from({ length: 25 }, (_, i) =>
      customer({ id: `customer-${i}`, first_name: `Customer${String(i).padStart(2, '0')}`, email: `c${i}@example.com` }))
    vi.mocked(customersApi.listCustomers).mockResolvedValue(many)

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await screen.findByText('Customer00 Doe')

    expect(screen.queryByText('Customer20 Doe')).not.toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByText('Customer20 Doe')).toBeInTheDocument()
    expect(screen.queryByText('Customer00 Doe')).not.toBeInTheDocument()
  })
})

describe('Customer detail page', () => {
  it('opens the detail view when clicking a row, and the edit form via its Edit button', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('jane@example.com')).toBeInTheDocument()
    expect(screen.queryByLabelText('Email')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(await screen.findByLabelText('Email')).toHaveValue('jane@example.com')
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('shows amount spent, order count, and a timeline entry per order', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001', total_minor_units: 5000, currency_code: 'USD' })],
      total: 1,
    })

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('$50.00')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText(/Placed order #1001/)).toBeInTheDocument()
  })

  it('shows the WhatsApp conversation panel with inbound/outbound messages when there are any', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(customersApi.listCustomerWhatsAppMessages).mockResolvedValue([
      { id: 'msg-2', phone_e164: '31612345678', direction: 'outbound', body: 'Widget - in stock: 7', created_at: '2026-08-23T10:01:00Z' },
      { id: 'msg-1', phone_e164: '31612345678', direction: 'inbound', body: 'STOCK ABC123', created_at: '2026-08-23T10:00:00Z' },
    ])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('WhatsApp conversation')).toBeInTheDocument()
    expect(screen.getByText('STOCK ABC123')).toBeInTheDocument()
    expect(screen.getByText('Widget - in stock: 7')).toBeInTheDocument()
    expect(customersApi.listCustomerWhatsAppMessages).toHaveBeenCalledWith('test-token', 'customer-1')
  })

  it('hides the WhatsApp conversation panel entirely when there is no history', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(customersApi.listCustomerWhatsAppMessages).mockResolvedValue([])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    await screen.findByText('jane@example.com')
    expect(screen.queryByText('WhatsApp conversation')).not.toBeInTheDocument()
  })

  // Reported: clicking the Orders stat should take you to that customer's
  // own orders, not be a dead number.
  it('links the Orders stat to that customer\'s filtered sales orders', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    await screen.findByText('Orders')
    expect(screen.getByRole('link', { name: /Orders/ })).toHaveAttribute('href', '/sales-orders?customer_id=customer-1')
  })

  it('shows "No orders yet" for Last order placed when the customer has none', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('No orders yet')).toBeInTheDocument()
  })

  it('shows a relative time for Last order placed when an order exists', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ occurred_at: twoHoursAgo })], total: 1,
    })

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('2 hours ago')).toBeInTheDocument()
  })

  // Reported bug: the customer page showed no order count, no amount
  // spent, and no timeline at all in production. Root cause: this fetch
  // requested perPage: 250, but ListSalesOrders' documented per_page
  // maximum is 200 -- the request 400'd, and the failure was silently
  // swallowed (empty stats, no error), which is why it took a devtools
  // Network-tab check to find. Pinning the exact param prevents a silent
  // regression back past the documented max.
  it('requests orders with a per_page within the documented maximum', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    await waitFor(() => expect(omsApi.listSalesOrders).toHaveBeenCalledWith('test-token', expect.objectContaining({
      customerId: 'customer-1',
    })))
    const perPage = vi.mocked(omsApi.listSalesOrders).mock.calls[0][1]?.perPage
    expect(perPage).toBeLessThanOrEqual(200)
  })

  it('surfaces an error instead of silently showing empty stats when the orders fetch fails', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(omsApi.listSalesOrders).mockRejectedValue(new Error('per_page too large'))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('per_page too large')).toBeInTheDocument()
  })

  it('editing the note saves it via update, preserving the rest of the profile', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(customersApi.updateCustomer).mockResolvedValue(customer({ customer_note: 'VIP' }))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))
    await screen.findByText('No notes')

    await userEvent.click(screen.getByRole('button', { name: 'Edit notes' }))
    await userEvent.type(await screen.findByLabelText('Notes'), 'VIP')
    await userEvent.click(screen.getByRole('button', { name: 'Save note' }))

    await waitFor(() => expect(customersApi.updateCustomer).toHaveBeenCalledWith('test-token', 'customer-1', expect.objectContaining({
      email: 'jane@example.com', first_name: 'Jane', last_name: 'Doe', customer_note: 'VIP',
    })))
  })
})

describe('Customer edit page', () => {

  it('opening Add Customer shows a blank create form', async () => {
    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))

    expect(await screen.findByText('New Customer', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create Customer' })).toBeInTheDocument()
  })

  it('creates a new customer and returns to the list', async () => {
    vi.mocked(customersApi.createCustomer).mockResolvedValue(
      customer({ id: 'customer-2', first_name: 'Bob', last_name: 'Smith', email: 'bob@example.com' }),
    )

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))
    await userEvent.type(await screen.findByLabelText('First name'), 'Bob')
    await userEvent.type(screen.getByLabelText('Last name'), 'Smith')
    await userEvent.type(screen.getByLabelText('Email'), 'bob@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Create Customer' }))

    await waitFor(() => expect(customersApi.createCustomer).toHaveBeenCalledWith('test-token', expect.objectContaining({
      email: 'bob@example.com', first_name: 'Bob', last_name: 'Smith',
    })))
    expect(await screen.findByText('Bob Smith')).toBeInTheDocument()
  })

  it('submits billing and shipping address fields', async () => {
    vi.mocked(customersApi.createCustomer).mockResolvedValue(
      customer({ id: 'customer-2', first_name: 'Bob', last_name: 'Smith', email: 'bob@example.com' }),
    )

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))
    await userEvent.type(await screen.findByLabelText('First name'), 'Bob')
    await userEvent.type(screen.getByLabelText('Last name'), 'Smith')
    await userEvent.type(screen.getByLabelText('Email'), 'bob@example.com')
    await userEvent.type(screen.getByLabelText('Billing city'), 'Rotterdam')
    await userEvent.click(screen.getByRole('button', { name: 'Create Customer' }))

    await waitFor(() => expect(customersApi.createCustomer).toHaveBeenCalledWith('test-token', expect.objectContaining({
      billing: expect.objectContaining({ city: 'Rotterdam' }),
    })))
  })

  it('shows an error if creating fails and stays on the form', async () => {
    vi.mocked(customersApi.createCustomer).mockRejectedValue(new Error('email already taken'))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))
    await userEvent.type(await screen.findByLabelText('Email'), 'bob@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Create Customer' }))

    expect(await screen.findByText('email already taken')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
  })

  it('renames a customer and returns to the detail view', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(customersApi.updateCustomer).mockResolvedValue(customer({ last_name: 'Doelist' }))

    const { router } = renderAt(<Customers />, '/customers')
    await userEvent.click(await screen.findByText('Jane Doe'))
    await userEvent.click(await screen.findByRole('button', { name: 'Edit' }))

    const input = await screen.findByLabelText('Last name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Doelist')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(customersApi.updateCustomer).toHaveBeenCalledWith(
      'test-token', 'customer-1', expect.objectContaining({ last_name: 'Doelist' }),
    ))
    // The edit form's own heading also shows the new name -- wait until
    // we're actually back on the detail URL before asserting on it.
    // (router state flips before React commits, so wait on the detail view's own UI)
    await screen.findByRole('button', { name: 'Edit' })
    expect(router.state.location.search).toBe('')
    expect(await screen.findByText('Jane Doelist', { selector: 'h1' })).toBeInTheDocument()
  })

  it('prefills fields when editing an existing customer', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer({
      billing: { city: 'Rotterdam', country: 'NL', phone: '+31 6 1234 5678' },
    })])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))
    await userEvent.click(await screen.findByRole('button', { name: 'Edit' }))

    expect(await screen.findByLabelText('Email')).toHaveValue('jane@example.com')
    expect(screen.getByLabelText('Billing city')).toHaveValue('Rotterdam')
    expect(screen.getByLabelText('Billing country')).toHaveValue('NL')
    expect(screen.getByLabelText('Billing phone')).toHaveValue('+31 6 1234 5678')
  })

  it('prefills fields when deep-linking straight to an existing customer\'s edit form, before the list has loaded', async () => {
    // Reproduces the reported bug: navigating directly to
    // /customers/<id>?edit=1 (global search, a direct URL, a hard refresh)
    // mounts CustomerEditPage before useCustomers' list fetch resolves, so
    // `customer` is null on first render. Without a hydration guard the
    // form's useState initializers never re-run once the real customer
    // arrives, leaving every field permanently blank.
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer({
      billing: { city: 'Rotterdam', country: 'NL', phone: '+31 6 1234 5678' },
    })])

    render(<MemoryRouter initialEntries={['/customers/customer-1?edit=1']}><Customers /></MemoryRouter>)

    expect(await screen.findByLabelText('Email')).toHaveValue('jane@example.com')
    expect(screen.getByLabelText('Billing city')).toHaveValue('Rotterdam')
    expect(screen.getByLabelText('Billing country')).toHaveValue('NL')
  })

  // Regression: global search's CommandPalette navigates to
  // "/customers/<id>" without unmounting whatever page is currently
  // showing. Guards against a future regression back toward local state
  // (see the identical Products.tsx bug this refactor fixed). It opens the
  // read-only detail view (same as any other row click), not the edit form.
  it('opens the customer detail view when navigated to its URL while already mounted on the list (search result click)', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(
      <MemoryRouter initialEntries={['/customers']}>
        <NavigateToCustomerButton id="customer-1" />
        <Customers />
      </MemoryRouter>,
    )
    await screen.findByText('Jane Doe')

    await userEvent.click(screen.getByText('simulate search result click'))

    expect(await screen.findByText('jane@example.com')).toBeInTheDocument()
    expect(screen.queryByLabelText('Email')).not.toBeInTheDocument()
  })

  it('Discard returns to the detail view without saving', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))
    await userEvent.click(await screen.findByRole('button', { name: 'Edit' }))
    await userEvent.type(await screen.findByLabelText('Last name'), ' extra')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(customersApi.updateCustomer).not.toHaveBeenCalled()
    expect(await screen.findByText('Jane Doe', { selector: 'h1' })).toBeInTheDocument()
  })
})

describe('Customers URLs', () => {
  it('page (?page=) is read from the URL', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue(Array.from({ length: 25 }, (_, i) => customer({ id: `c${i}`, email: `c${i}@x.com` })))
    renderAt(<Customers />, '/customers', '/customers?page=2')
    expect(await screen.findByText('Page 2 of 2')).toBeInTheDocument()
  })

  it('closing a customer returns to the list page it was opened from', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    const { router } = renderAt(<Customers />, '/customers', '/customers?q=a')
    const c = customer()
    await userEvent.click(await screen.findByText(`${c.first_name} ${c.last_name}`))
    expect(router.state.location.pathname).toBe(`/customers/${c.id}`)
    await act(() => router.navigate(-1))
    expect(router.state.location.pathname + router.state.location.search).toBe('/customers?q=a')
  })

  it('an unknown id in edit mode shows not-found instead of a blank create form', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    renderAt(<Customers />, '/customers', '/customers/nope?edit=1')
    expect(await screen.findByText('Customer not found')).toBeInTheDocument()
  })
})
