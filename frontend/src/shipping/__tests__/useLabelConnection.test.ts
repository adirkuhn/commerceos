// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useLabelConnection } from '../useLabelConnection'
import { shippingConnectionsOf } from '../../test/shippingConnections'

vi.mock('../api', () => ({ listShippingConnections: vi.fn(), updateShippingPreferences: vi.fn(), getShippingSettings: vi.fn() }))
import * as shippingApi from '../api'

type Settings = Awaited<ReturnType<typeof shippingApi.getShippingSettings>>
const settings = (defaults: Record<string, unknown>) => ({ settings: defaults, carrier_overrides: [] }) as unknown as Settings

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(
    shippingConnectionsOf([{ id: 'mp', provider: 'myparcel' }, { id: 'pn', provider: 'postnl' }], 'pn'),
  )
  vi.mocked(shippingApi.getShippingSettings).mockImplementation(async (_token, id) => id === 'mp'
    ? settings({ default_carrier: 4, default_package_type: 2, default_label_size: 'A6', auto_print_enabled: true })
    : settings({ default_carrier: 1, default_package_type: 1, default_label_size: 'A4', auto_print_enabled: false }))
})

async function setup() {
  const hook = renderHook(() => useLabelConnection('t'))
  await waitFor(() => expect(hook.result.current.connection).not.toBeNull())
  await waitFor(() => expect(shippingApi.getShippingSettings).toHaveBeenCalled())
  return hook
}

describe('useLabelConnection', () => {
  it('starts on the default connection and loads its settings', async () => {
    const { result } = await setup()
    expect(result.current.connection?.id).toBe('pn')
    expect(shippingApi.getShippingSettings).toHaveBeenCalledWith('t', 'pn')
    expect(result.current.capabilities.hasCarrierChoice).toBe(false)
  })

  it('switching loads the chosen connection settings', async () => {
    const { result } = await setup()
    act(() => result.current.selectConnection('mp'))
    expect(result.current.connection?.id).toBe('mp')
    await waitFor(() => expect(result.current.defaultCarrier).toBe(4))
    expect(result.current.defaultPackageType).toBe(2)
    expect(result.current.labelSize).toBe('A6')
    expect(result.current.autoPrintEnabled).toBe(true)
    expect(result.current.capabilities.hasCarrierChoice).toBe(true)
  })

  it('a carrier choice is only used by a provider that has one', async () => {
    const { result } = await setup()
    expect(result.current.carrierFor(9)).toBe(1)
    act(() => result.current.selectConnection('mp'))
    await waitFor(() => expect(result.current.defaultCarrier).toBe(4))
    expect(result.current.carrierFor(9)).toBe(9)
    expect(result.current.carrierFor(null)).toBe(4)
  })

  it('settings that fail to load keep the built-in defaults', async () => {
    vi.mocked(shippingApi.getShippingSettings).mockRejectedValue(new Error('none'))
    const { result } = await setup()
    expect(result.current.defaultCarrier).toBe(1)
    expect(result.current.labelSize).toBe('A4')
    expect(result.current.autoPrintEnabled).toBe(false)
  })

  it('without connections there is nothing to load', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
    const { result } = renderHook(() => useLabelConnection('t'))
    await waitFor(() => expect(shippingApi.listShippingConnections).toHaveBeenCalled())
    await act(async () => {})
    expect(result.current.connection).toBeNull()
    expect(shippingApi.getShippingSettings).not.toHaveBeenCalled()
  })
})
