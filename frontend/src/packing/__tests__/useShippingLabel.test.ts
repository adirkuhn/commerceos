// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useShippingLabel } from '../useShippingLabel'
import type { SalesOrder } from '../../oms/types'
import type { RecipientAddress, Shipment } from '../../shipping/types'

vi.mock('../../shipping/api', () => ({ listShippingConnections: vi.fn(), getShippingSettings: vi.fn(), listShipmentsForOrder: vi.fn(), createShipment: vi.fn() }))
vi.mock('../../shipping/openLabel', () => ({ loadShipmentLabelIntoTab: vi.fn(), openShipmentLabel: vi.fn() }))

import { shippingConnectionsOf } from '../../test/shippingConnections'
import * as shippingApi from '../../shipping/api'
import * as openLabel from '../../shipping/openLabel'

const order = { id: 'so-1' } as SalesOrder
const recipient = { cc: 'NL', city: 'Amsterdam' } as RecipientAddress

// Waits for the connection defaults, so later updates land inside act.
async function setup(beforeCreate = vi.fn().mockResolvedValue(true), onError = vi.fn()) {
  const hook = renderHook(() => useShippingLabel({ token: 't', activeOrder: order, beforeCreate, onError }))
  await waitFor(() => expect(hook.result.current.labelSize).toBe('A6'))
  return { ...hook, beforeCreate, onError }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'c1', provider: 'myparcel' }]))
  vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: { auto_print_enabled: true, default_label_size: 'A6', default_carrier: 4, default_package_type: 2 } } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
  vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
})

describe('useShippingLabel', () => {
  it('loads the connection defaults', async () => {
    const { result } = await setup()
    await waitFor(() => expect(result.current.autoPrintEnabled).toBe(true))
    expect(result.current.labelSize).toBe('A6')
    expect(result.current.shipping.defaultCarrier).toBe(4)
    expect(result.current.shipping.defaultPackageType).toBe(2)
    expect(result.current.shippingConnection?.id).toBe('c1')
  })

  it('does nothing when the caller refuses the claim', async () => {
    const { result } = await setup(vi.fn().mockResolvedValue(false))
    await act(() => result.current.createLabel())
    expect(shippingApi.listShipmentsForOrder).not.toHaveBeenCalled()
    expect(result.current.labelStep).toBeNull()
  })

  it('an active shipment asks first; reprinting opens it and remembers it', async () => {
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([
      { id: 'old', cancelled_at: '2026-01-01' } as Shipment, { id: 'sh-1', cancelled_at: null } as Shipment,
    ])
    vi.mocked(openLabel.openShipmentLabel).mockResolvedValue(null)
    const { result } = await setup()
    await act(() => result.current.createLabel())
    expect(result.current.existingShipment?.id).toBe('sh-1')

    await act(async () => { result.current.reprintExisting() })
    expect(result.current.existingShipment).toBeNull()
    expect(result.current.createdShipmentId).toBe('sh-1')
    expect(openLabel.openShipmentLabel).toHaveBeenCalledWith('t', 'sh-1', 'A6')
  })

  it('creating a new one instead goes to the address step', async () => {
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([{ id: 'sh-1', cancelled_at: null } as Shipment])
    const { result } = await setup()
    await act(() => result.current.createLabel())
    act(() => result.current.createNewInsteadOfExisting())
    expect(result.current.existingShipment).toBeNull()
    expect(result.current.labelStep).toBe('address')
  })

  it('address -> preview -> print creates the shipment and loads its label', async () => {
    const tab = { close: vi.fn() } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(tab)
    vi.mocked(shippingApi.createShipment).mockResolvedValue({ id: 'new-sh' } as Shipment)
    vi.mocked(openLabel.loadShipmentLabelIntoTab).mockResolvedValue(tab)
    const { result } = await setup()
    await act(() => result.current.createLabel())
    expect(result.current.labelStep).toBe('address')

    act(() => result.current.confirmAddress({ carrier: 1, packageType: 3, recipient }))
    expect(result.current.labelStep).toBe('preview')

    await act(() => result.current.printLabel())
    expect(shippingApi.createShipment).toHaveBeenCalledWith('t', { carrier: 1, package_type: 3, recipient, connection_id: 'c1', sales_order_id: 'so-1' })
    expect(openLabel.loadShipmentLabelIntoTab).toHaveBeenCalledWith(tab, 't', 'new-sh', 'A6')
    expect(result.current.labelStep).toBeNull()
    expect(result.current.createdShipmentId).toBe('new-sh')
    expect(result.current.labelTab).toBe(tab)
  })

  it('a failed shipment closes the pre-opened tab and rethrows', async () => {
    const tab = { close: vi.fn() } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(tab)
    vi.mocked(shippingApi.createShipment).mockRejectedValue(new Error('carrier down'))
    const { result } = await setup()
    act(() => result.current.confirmAddress({ carrier: 1, packageType: 1, recipient }))
    await act(() => expect(result.current.printLabel()).rejects.toThrow('carrier down'))
    expect(tab.close).toHaveBeenCalled()
    expect(result.current.labelStep).toBe('preview')
  })

  it('a label that fails to open is reported', async () => {
    vi.mocked(openLabel.openShipmentLabel).mockRejectedValue(new Error('no label'))
    const { result, onError } = await setup()
    await act(() => result.current.openLabel('sh-1'))
    expect(onError).toHaveBeenCalledWith('no label')
  })

  it('clear drops the half-done flow', async () => {
    const { result } = await setup()
    await act(() => result.current.createLabel())
    act(() => result.current.clear())
    expect(result.current.labelStep).toBeNull()
  })

  it('a label made after switching provider goes through the chosen connection', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'c1', provider: 'myparcel' }, { id: 'c2', provider: 'postnl' }]))
    vi.spyOn(window, 'open').mockReturnValue(null)
    vi.mocked(shippingApi.createShipment).mockResolvedValue({ id: 'new-sh' } as Shipment)
    const { result } = await setup()
    act(() => result.current.shipping.selectConnection('c2'))
    await waitFor(() => expect(shippingApi.getShippingSettings).toHaveBeenLastCalledWith('t', 'c2'))
    // A carrier picked for MyParcel is not sent to a provider that is its
    // own carrier -- the connection's default goes instead.
    act(() => result.current.confirmAddress({ carrier: 9, packageType: 1, recipient }))
    await act(() => result.current.printLabel())
    expect(shippingApi.createShipment).toHaveBeenCalledWith('t', expect.objectContaining({ connection_id: 'c2', carrier: 4 }))
  })
})
