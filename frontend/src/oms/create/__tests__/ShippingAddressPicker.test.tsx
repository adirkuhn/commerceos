// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

vi.mock('../../../customers/api', () => ({
  listShippingAddresses: vi.fn(), addShippingAddress: vi.fn(), updateShippingAddress: vi.fn(),
  removeShippingAddress: vi.fn(), setDefaultShippingAddress: vi.fn(),
}))
import * as api from '../../../customers/api'
import { ShippingAddressPicker } from '../ShippingAddressPicker'
import type { ShippingAddress } from '../../../customers/types'

const main: ShippingAddress = {
  id: 'sa-1', customer_id: 'c-1', label: 'Main', first_name: 'Ana', last_name: '', company: '', address_1: 'Main 1', address_2: '',
  city: 'Utrecht', state: '', postcode: '3511', country: 'NL', phone: '', is_default: true,
}
const dock: ShippingAddress = { ...main, id: 'sa-2', label: 'Dock', address_1: 'Dock 2', is_default: false }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listShippingAddresses).mockResolvedValue([main, dock])
})

describe('ShippingAddressPicker', () => {
  it('offers the customer\'s addresses, the one in use selected, and picks another', async () => {
    const onPick = vi.fn()
    render(<ShippingAddressPicker token="tok" customerId="c-1" value={{ first_name: 'Ana', address_1: 'Main 1', city: 'Utrecht', postcode: '3511', country: 'NL' }} onPick={onPick} />)

    const select = await screen.findByRole('combobox', { name: 'Ship to' })
    expect(select).toHaveValue('sa-1')
    expect(screen.getByRole('option', { name: 'Dock — Dock 2, 3511 Utrecht, NL' })).toBeInTheDocument()

    fireEvent.change(select, { target: { value: 'sa-2' } })
    expect(onPick).toHaveBeenCalledWith(expect.objectContaining({ address_1: 'Dock 2', city: 'Utrecht', first_name: 'Ana' }))
    expect(onPick.mock.calls[0][0]).not.toHaveProperty('label')
  })

  it('shows no choice for an address typed by hand', async () => {
    render(<ShippingAddressPicker token="tok" customerId="c-1" value={{ address_1: 'Elsewhere 9' }} onPick={vi.fn()} />)
    expect(await screen.findByRole('combobox', { name: 'Ship to' })).toHaveValue('')
  })

  it('stays hidden with one address or no customer', async () => {
    vi.mocked(api.listShippingAddresses).mockResolvedValue([main])
    const { container, rerender } = render(<ShippingAddressPicker token="tok" customerId="c-1" value={{}} onPick={vi.fn()} />)
    await vi.waitFor(() => expect(api.listShippingAddresses).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()

    rerender(<ShippingAddressPicker token="tok" customerId={null} value={{}} onPick={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
  })
})
