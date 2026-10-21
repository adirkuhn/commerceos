// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ShippingLabelFlow } from '../ShippingLabelFlow'
import { useShippingLabel } from '../useShippingLabel'
import type { SalesOrder } from '../../oms/types'
import type { Shipment } from '../../shipping/types'
import { shippingConnectionsOf } from '../../test/shippingConnections'

vi.mock('../../shipping/api', () => ({
  listShippingConnections: vi.fn(), getShippingSettings: vi.fn(), listShipmentsForOrder: vi.fn(), createShipment: vi.fn(),
  lookupAddress: vi.fn(), getDeliveryPriceEstimate: vi.fn(),
}))
vi.mock('../../shipping/openLabel', () => ({ loadShipmentLabelIntoTab: vi.fn(), openShipmentLabel: vi.fn() }))
import * as shippingApi from '../../shipping/api'

const order = {
  id: 'so-1', shipping_method: 'flat_rate', shipping_method_title: 'DPD',
  shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Hauptstrasse 1', city: 'Berlin', postcode: '10115', country: 'DE', email: 'jane@example.com' },
} as unknown as SalesOrder

function Harness() {
  const label = useShippingLabel({ token: 't', activeOrder: order, beforeCreate: async () => true, onError: vi.fn() })
  return (
    <>
      <button type="button" disabled={!label.shippingConnection} onClick={label.createLabel}>start</button>
      <ShippingLabelFlow token="t" activeOrder={order} label={label} />
    </>
  )
}

async function openAddressStep() {
  render(<MemoryRouter><Harness /></MemoryRouter>)
  const start = screen.getByRole('button', { name: 'start' })
  await waitFor(() => expect(start).toBeEnabled())
  await userEvent.click(start)
  await screen.findByText('Confirm shipping address')
}

async function printLabel() {
  await userEvent.click(screen.getByRole('button', { name: 'Review label' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Print label' }))
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
  vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
  vi.mocked(shippingApi.getDeliveryPriceEstimate).mockResolvedValue([])
  vi.mocked(shippingApi.createShipment).mockResolvedValue({ id: 'sh-1' } as Shipment)
  vi.spyOn(window, 'open').mockReturnValue(null)
})

describe('ShippingLabelFlow provider', () => {
  it('with one connection it names the provider, no dropdown', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'mp', provider: 'myparcel' }]))
    await openAddressStep()
    expect(screen.getByText('Shipping via MyParcel')).toBeInTheDocument()
    expect(screen.queryByLabelText('Shipping provider')).not.toBeInTheDocument()
    // MyParcel keeps its carrier choice, guessed from the shipping method.
    expect(screen.getByLabelText('Carrier')).toHaveValue('4')
  })

  it('preselects the default and sends the connection picked for this label', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(
      shippingConnectionsOf([{ id: 'mp', provider: 'myparcel' }, { id: 'pn', provider: 'postnl' }], 'pn'),
    )
    await openAddressStep()
    const provider = screen.getByLabelText('Shipping provider')
    expect(provider).toHaveValue('pn')
    expect(screen.queryByLabelText('Carrier')).not.toBeInTheDocument()

    await userEvent.selectOptions(provider, 'mp')
    expect(screen.getByLabelText('Carrier')).toBeInTheDocument()
    await printLabel()

    await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith('t', expect.objectContaining({ connection_id: 'mp', carrier: 4 })))
  })

  it('a provider that is its own carrier sends its default carrier', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(
      shippingConnectionsOf([{ id: 'mp', provider: 'myparcel' }, { id: 'pn', provider: 'postnl' }]),
    )
    await openAddressStep()
    await userEvent.selectOptions(screen.getByLabelText('Shipping provider'), 'pn')
    expect(screen.queryByLabelText('Carrier')).not.toBeInTheDocument()
    await printLabel()

    await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith('t', expect.objectContaining({ connection_id: 'pn', carrier: 1 })))
  })

  it('without a shipping connection no label can start', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
    render(<Harness />)
    await waitFor(() => expect(shippingApi.listShippingConnections).toHaveBeenCalled())
    expect(screen.getByRole('button', { name: 'start' })).toBeDisabled()
  })
})

describe('ShippingLabelFlow readiness', () => {
  it('warns and blocks the label while the connection misses a sender address; another connection still works', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf(
      [{ id: 'mp', provider: 'myparcel' }, { id: 'pn', provider: 'postnl', missing: ['sender_address'] }], 'pn',
    ))
    await openAddressStep()
    expect(screen.getByText('PostNL needs a sender address before it can create labels.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Set up the shipping addresses' })).toHaveAttribute('href', '/integrations#shipping-addresses')
    expect(screen.getByRole('button', { name: 'Review label' })).toBeDisabled()

    await userEvent.selectOptions(screen.getByLabelText('Shipping provider'), 'mp')
    expect(screen.queryByText('PostNL needs a sender address before it can create labels.')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Review label' })).toBeEnabled()
  })
})
