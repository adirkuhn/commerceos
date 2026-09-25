// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { usePackingLabel } from '../usePackingLabel'
import type { SalesOrder } from '../../oms/types'
import type { Shipment, RecipientAddress } from '../../shipping/types'

vi.mock('../api', () => ({ touchPackingSession: vi.fn() }))
vi.mock('../../integrations/hooks/useConnections', () => ({ useConnections: vi.fn() }))
vi.mock('../../shipping/api', () => ({ getShippingSettings: vi.fn(), listShipmentsForOrder: vi.fn(), createShipment: vi.fn() }))
vi.mock('../../shipping/openLabel', () => ({ openShipmentLabel: vi.fn(), loadShipmentLabelIntoTab: vi.fn() }))

import * as api from '../api'
import * as shippingApi from '../../shipping/api'
import { openShipmentLabel, loadShipmentLabelIntoTab } from '../../shipping/openLabel'
import { useConnections } from '../../integrations/hooks/useConnections'

type Connection = ReturnType<typeof useConnections>['connections'][number]
const MYPARCEL = [{ id: 'c1', provider: 'myparcel' } as Connection]
const order = { id: 'so-1' } as SalesOrder
const recipient = { name: 'Jane' } as unknown as RecipientAddress
const shipment = (overrides: Partial<Shipment> = {}) => ({ id: 'sh-1', cancelled_at: null, ...overrides }) as Shipment

function withConnections(connections: Connection[]) {
  vi.mocked(useConnections).mockReturnValue({ connections, loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
}

let reload: ReturnType<typeof vi.fn>
let onAuthError: ReturnType<typeof vi.fn>
let setError: ReturnType<typeof vi.fn>

function setup(finished = false) {
  return renderHook(() => usePackingLabel({ token: 't', sessionId: 'pack-1', order, finished, reload, onAuthError, setError }))
}

beforeEach(() => {
  vi.clearAllMocks()
  reload = vi.fn().mockResolvedValue(undefined)
  onAuthError = vi.fn()
  setError = vi.fn()
  withConnections(MYPARCEL)
  vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: { auto_print_enabled: true, default_label_size: 'A6', default_carrier: 2, default_package_type: 3 } } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
  vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
  vi.mocked(api.touchPackingSession).mockResolvedValue(null)
})

describe('usePackingLabel settings', () => {
  it('applies the connection\'s label defaults', async () => {
    const { result } = setup()
    await waitFor(() => expect(result.current.labelSize).toBe('A6'))
    expect(result.current.autoPrintEnabled).toBe(true)
    expect(result.current.connectionDefaultCarrier).toBe(2)
    expect(result.current.connectionDefaultPackageType).toBe(3)
  })

  it('without a MyParcel connection does nothing', async () => {
    withConnections([])
    const { result } = setup()
    await act(async () => { await result.current.createLabel() })
    expect(shippingApi.getShippingSettings).not.toHaveBeenCalled()
    expect(result.current.labelStep).toBeNull()
  })
})

describe('usePackingLabel creating', () => {
  it('before finishing it takes the lock, then opens the address step', async () => {
    const { result } = setup()
    await act(async () => { await result.current.createLabel() })
    expect(api.touchPackingSession).toHaveBeenCalledWith('t', 'pack-1')
    expect(reload).toHaveBeenCalled()
    expect(result.current.labelStep).toBe('address')
  })

  it('after finishing it skips the lock', async () => {
    const { result } = setup(true)
    await act(async () => { await result.current.createLabel() })
    expect(api.touchPackingSession).not.toHaveBeenCalled()
    expect(result.current.labelStep).toBe('address')
  })

  it('a lock held elsewhere stops it with a message', async () => {
    vi.mocked(api.touchPackingSession).mockRejectedValue('locked')
    const { result } = setup()
    await act(async () => { await result.current.createLabel() })
    expect(setError).toHaveBeenCalledWith('Someone else is packing this order')
    expect(result.current.labelStep).toBeNull()
  })

  it('an order that already has a live shipment asks first', async () => {
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([shipment({ id: 'old', cancelled_at: '2026-01-01' }), shipment()])
    const { result } = setup()
    await act(async () => { await result.current.createLabel() })
    expect(result.current.existingShipment?.id).toBe('sh-1')
    expect(result.current.labelStep).toBeNull()

    act(() => result.current.createNewOverExisting())
    expect(result.current.existingShipment).toBeNull()
    expect(result.current.labelStep).toBe('address')
  })

  it('reprinting the existing shipment opens its label', async () => {
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([shipment()])
    const tab = {} as Window
    vi.mocked(openShipmentLabel).mockResolvedValue(tab)
    const { result } = setup()
    await waitFor(() => expect(result.current.labelSize).toBe('A6'))
    await act(async () => { await result.current.createLabel() })

    act(() => result.current.reprintExisting())

    await waitFor(() => expect(result.current.labelTab).toBe(tab))
    expect(openShipmentLabel).toHaveBeenCalledWith('t', 'sh-1', 'A6')
    expect(result.current.createdShipmentId).toBe('sh-1')
    expect(result.current.existingShipment).toBeNull()
  })

  it('confirming the address moves to preview, printing creates the shipment', async () => {
    const tab = { close: vi.fn() } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(tab)
    vi.mocked(shippingApi.createShipment).mockResolvedValue(shipment({ id: 'new' }))
    vi.mocked(loadShipmentLabelIntoTab).mockResolvedValue(tab)
    const { result } = setup()
    await waitFor(() => expect(result.current.labelSize).toBe('A6'))

    act(() => result.current.confirmAddress({ carrier: 5, packageType: 1, recipient }))
    expect(result.current.labelStep).toBe('preview')

    await act(async () => { await result.current.printLabel() })

    expect(shippingApi.createShipment).toHaveBeenCalledWith('t', { carrier: 5, package_type: 1, recipient, connection_id: 'c1', sales_order_id: 'so-1' })
    expect(result.current.labelStep).toBeNull()
    expect(result.current.createdShipmentId).toBe('new')
    expect(result.current.labelTab).toBe(tab)
  })

  it('a failed shipment closes the pre-opened tab and rethrows', async () => {
    const tab = { close: vi.fn() } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(tab)
    vi.mocked(shippingApi.createShipment).mockRejectedValue(new Error('Bad address'))
    const { result } = setup()
    act(() => result.current.confirmAddress({ carrier: 1, packageType: 1, recipient }))

    await act(async () => { await expect(result.current.printLabel()).rejects.toThrow('Bad address') })
    expect(tab.close).toHaveBeenCalled()
    expect(result.current.labelStep).toBe('preview')
  })
})
