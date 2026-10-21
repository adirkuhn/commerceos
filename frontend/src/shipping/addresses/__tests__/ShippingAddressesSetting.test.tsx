// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ShippingAddressesSetting } from '../ShippingAddressesSetting'
import { CodedApiError } from '../../../api/codedApiError'
import { shippingConnectionsOf } from '../../../test/shippingConnections'
import type { ShippingAddresses } from '../../types'

vi.mock('../../api', () => ({
  listShippingConnections: vi.fn(), updateShippingPreferences: vi.fn(),
  getShippingAddresses: vi.fn(), updateShippingAddresses: vi.fn(),
}))
import * as shippingApi from '../../api'

const company = { company: 'Shop BV', street: 'Damrak', number: '1', postal_code: '1012LG', city: 'Amsterdam', cc: 'NL' }
const warehouse = { company: 'Shop BV Depot', street: 'Kade', number: '9', number_suffix: 'B', postal_code: '3011AA', city: 'Rotterdam', cc: 'NL' }

function addresses(overrides: Partial<ShippingAddresses> = {}): ShippingAddresses {
  return { sender_address: null, return_address: null, company_address: company, ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'pn', provider: 'postnl' }]))
  vi.mocked(shippingApi.getShippingAddresses).mockResolvedValue(addresses())
  vi.mocked(shippingApi.updateShippingAddresses).mockImplementation(async (_t, input) => addresses(input))
})

async function renderSetting(canEdit = true) {
  render(<MemoryRouter><ShippingAddressesSetting token="t" canEdit={canEdit} /></MemoryRouter>)
  const region = await screen.findByRole('region', { name: 'Shipping addresses' })
  await screen.findByRole('group', { name: 'Sender address' })
  return region
}

const sender = () => screen.getByRole('group', { name: 'Sender address' })
const returns = () => screen.getByRole('group', { name: 'Return address' })

async function fill(group: HTMLElement, a: typeof warehouse) {
  const g = within(group)
  await userEvent.type(g.getByLabelText('Company'), a.company)
  await userEvent.type(g.getByLabelText('Street'), a.street)
  await userEvent.type(g.getByLabelText('House number'), a.number)
  if (a.number_suffix) await userEvent.type(g.getByLabelText('Addition'), a.number_suffix)
  await userEvent.type(g.getByLabelText('Postal code'), a.postal_code)
  await userEvent.type(g.getByLabelText('City'), a.city)
  await userEvent.selectOptions(g.getByLabelText('Country'), a.cc)
}

describe('ShippingAddressesSetting', () => {
  it('defaults to the company profile address and same-as-sender return', async () => {
    await renderSetting()
    expect(within(sender()).getByLabelText('Use the company profile address')).toBeChecked()
    expect(within(sender()).getByText(/Damrak 1/)).toBeInTheDocument()
    expect(within(returns()).getByLabelText('Same as the sender')).toBeChecked()
  })

  it('an admin saving the defaults sends null for both', async () => {
    await renderSetting()
    await userEvent.click(screen.getByRole('button', { name: 'Save addresses' }))
    expect(shippingApi.updateShippingAddresses).toHaveBeenCalledWith('t', { sender_address: null, return_address: null })
    expect(await screen.findByText('Shipping addresses saved.')).toBeInTheDocument()
  })

  it('an admin saves a different sender and return address', async () => {
    await renderSetting()
    await userEvent.click(within(sender()).getByLabelText('Use a different address'))
    await fill(sender(), warehouse)
    await userEvent.click(within(returns()).getByLabelText('Use a different return address'))
    await fill(returns(), { ...warehouse, number_suffix: '', company: 'Returns' })
    await userEvent.click(screen.getByRole('button', { name: 'Save addresses' }))
    expect(shippingApi.updateShippingAddresses).toHaveBeenCalledWith('t', {
      sender_address: warehouse,
      return_address: { company: 'Returns', street: 'Kade', number: '9', postal_code: '3011AA', city: 'Rotterdam', cc: 'NL' },
    })
  })

  it('loads saved addresses into the forms', async () => {
    vi.mocked(shippingApi.getShippingAddresses).mockResolvedValue(addresses({ sender_address: warehouse, return_address: null }))
    await renderSetting()
    expect(within(sender()).getByLabelText('Use a different address')).toBeChecked()
    expect(within(sender()).getByLabelText('City')).toHaveValue('Rotterdam')
    expect(within(returns()).getByLabelText('Same as the sender')).toBeChecked()
  })

  it('flags missing required fields instead of saving', async () => {
    await renderSetting()
    await userEvent.click(within(sender()).getByLabelText('Use a different address'))
    await userEvent.type(within(sender()).getByLabelText('Company'), 'Shop')
    await userEvent.click(screen.getByRole('button', { name: 'Save addresses' }))
    expect(shippingApi.updateShippingAddresses).not.toHaveBeenCalled()
    expect(within(sender()).getByLabelText('Street')).toHaveAttribute('aria-invalid', 'true')
    expect(within(sender()).getAllByText('Required').length).toBeGreaterThan(0)
    expect(within(sender()).getByLabelText('Company')).not.toHaveAttribute('aria-invalid')
  })

  it('without a company address it says so and links to the company profile', async () => {
    vi.mocked(shippingApi.getShippingAddresses).mockResolvedValue(addresses({ company_address: null }))
    await renderSetting()
    expect(within(sender()).getByText(/company profile has no address yet/)).toBeInTheDocument()
    expect(within(sender()).getByRole('link', { name: 'Fill in the company profile' })).toHaveAttribute('href', '/settings')
  })

  it('shows why a save failed', async () => {
    vi.mocked(shippingApi.updateShippingAddresses).mockRejectedValue(new CodedApiError('SHIPPING_ADDRESS_INVALID', 'invalid', undefined))
    await renderSetting()
    await userEvent.click(screen.getByRole('button', { name: 'Save addresses' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('A shipping address needs a company name')
    expect(screen.queryByText('Shipping addresses saved.')).not.toBeInTheDocument()
  })

  it('refreshes the connections after saving so readiness updates', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(
      shippingConnectionsOf([{ id: 'pn', provider: 'postnl', missing: ['sender_address'] }]),
    )
    vi.mocked(shippingApi.getShippingAddresses).mockResolvedValue(addresses({ company_address: null }))
    await renderSetting()
    expect(screen.getByText('PostNL needs a sender address before it can create labels.')).toBeInTheDocument()

    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'pn', provider: 'postnl' }]))
    await userEvent.click(within(sender()).getByLabelText('Use a different address'))
    await fill(sender(), warehouse)
    await userEvent.click(screen.getByRole('button', { name: 'Save addresses' }))
    await waitFor(() => expect(screen.queryByText('PostNL needs a sender address before it can create labels.')).not.toBeInTheDocument())
    expect(shippingApi.listShippingConnections).toHaveBeenCalledTimes(2)
  })

  it('is read-only for a non-admin', async () => {
    vi.mocked(shippingApi.getShippingAddresses).mockResolvedValue(addresses({ sender_address: warehouse }))
    await renderSetting(false)
    expect(within(sender()).getByLabelText('Use the company profile address')).toBeDisabled()
    expect(within(sender()).getByLabelText('City')).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Save addresses' })).not.toBeInTheDocument()
    expect(screen.getByText('Only an owner or admin can change this.')).toBeInTheDocument()
  })

  it('shows a load failure', async () => {
    vi.mocked(shippingApi.getShippingAddresses).mockRejectedValue(new Error('down'))
    render(<MemoryRouter><ShippingAddressesSetting token="t" canEdit /></MemoryRouter>)
    expect(await screen.findByText('down')).toBeInTheDocument()
  })

  it('is hidden without shipping connections', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
    const { container } = render(<MemoryRouter><ShippingAddressesSetting token="t" canEdit /></MemoryRouter>)
    await waitFor(() => expect(shippingApi.listShippingConnections).toHaveBeenCalled())
    await waitFor(() => expect(container).toBeEmptyDOMElement())
  })
})
