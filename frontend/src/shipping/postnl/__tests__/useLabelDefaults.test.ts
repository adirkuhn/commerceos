// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useLabelDefaults } from '../useLabelDefaults'
import type { ShippingConnectionSettings } from '../../types'

vi.mock('../../api', () => ({ getShippingSettings: vi.fn(), updateShippingSettings: vi.fn() }))
import * as shippingApi from '../../api'

const saved: ShippingConnectionSettings = {
  connection_id: 'pn', default_carrier: 1, default_package_type: 2, default_label_format: 'pdf', default_label_size: 'A6',
  default_signature_required: false, default_only_recipient: false, insure_by_default: false,
  myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: false, export_order_to_myparcel: false,
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: saved, carrier_overrides: [] })
})

describe('useLabelDefaults', () => {
  it('loads the connection settings', async () => {
    const { result } = renderHook(() => useLabelDefaults('t', 'pn'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.settings).toEqual(saved)
  })

  it('shows why loading failed', async () => {
    vi.mocked(shippingApi.getShippingSettings).mockRejectedValue(new Error('down'))
    const { result } = renderHook(() => useLabelDefaults('t', 'pn'))
    await waitFor(() => expect(result.current.error).toBe('down'))
  })

  it('saves the full settings object with the shown fields changed', async () => {
    vi.mocked(shippingApi.updateShippingSettings).mockImplementation(async (_t, id, input) => ({ connection_id: id, ...input }))
    const { result } = renderHook(() => useLabelDefaults('t', 'pn'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    await act(() => result.current.save({
      default_label_format: 'zpl', default_package_type: 6, default_signature_required: true,
      default_only_recipient: true, insure_by_default: false, auto_print_enabled: true,
    }))
    expect(shippingApi.updateShippingSettings).toHaveBeenCalledWith('t', 'pn', {
      default_carrier: 1, default_package_type: 6, default_label_format: 'zpl', default_label_size: 'A6',
      default_signature_required: true, default_only_recipient: true, insure_by_default: false,
      myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: true, export_order_to_myparcel: false,
    })
    expect(result.current.settings?.default_package_type).toBe(6)
  })
})
