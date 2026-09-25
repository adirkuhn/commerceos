// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePackActions } from '../usePackActions'
import type { PackingSessionDetail, PickingSessionLineItem } from '../../scanner/types'

vi.mock('../../scanner/api', () => ({
  scanPackItem: vi.fn(), resolvePackScanCode: vi.fn(), substitutePackLine: vi.fn(), refundPackLine: vi.fn(), deletePackScan: vi.fn(),
}))
vi.mock('../../scanner/errorSound', () => ({ playErrorSound: vi.fn() }))
import * as packingApi from '../../scanner/api'
import { playErrorSound } from '../../scanner/errorSound'

const line: PickingSessionLineItem = { id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'W', barcode: '', quantity: 2, price_minor_units: 0, total_minor_units: 0 }
const scan = { id: 's1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack' as const, quantity: 1, scanned_at: '', scanned_by: '' }
const session = { session: { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' }, line_items: [line], scans: [] } as PackingSessionDetail

function setup() {
  const opts = { token: 't', session, loadSession: vi.fn().mockResolvedValue(undefined), setError: vi.fn() }
  return { ...renderHook(() => usePackActions(opts)), opts }
}

beforeEach(() => vi.clearAllMocks())

describe('usePackActions', () => {
  it('a mis-scan raises the blocking alert and keeps the code', async () => {
    vi.mocked(packingApi.scanPackItem).mockRejectedValue(new Error('not on order'))
    const { result } = setup()
    act(() => result.current.setCode(' 999 '))
    await act(() => result.current.submitScan())
    expect(packingApi.scanPackItem).toHaveBeenCalledWith('t', 'pack-1', { code: '999', quantity: 1 })
    expect(playErrorSound).toHaveBeenCalled()
    expect(result.current.scanErrorAlert).toBe('not on order')
    expect(result.current.code).toBe(' 999 ')

    // No further scans until acknowledged.
    await act(() => result.current.submitScan())
    expect(packingApi.scanPackItem).toHaveBeenCalledTimes(1)
    act(() => result.current.dismissScanError())
    expect(result.current.scanErrorAlert).toBeNull()
  })

  it('a good scan clears the code and reloads', async () => {
    vi.mocked(packingApi.scanPackItem).mockResolvedValue({ scan })
    const { result, opts } = setup()
    act(() => result.current.setCode('111'))
    await act(() => result.current.submitScan())
    expect(result.current.code).toBe('')
    expect(opts.loadSession).toHaveBeenCalledWith('pack-1')
  })

  it('lines without a SKU do not open; opening a line closes add-item and vice versa', () => {
    const { result } = setup()
    act(() => result.current.openLine({ ...line, sku_id: '' }, 0))
    expect(result.current.packLine).toBeNull()

    act(() => result.current.openAddItem())
    act(() => result.current.openLine(line, 1))
    expect(result.current.addItemOpen).toBe(false)
    expect(result.current.packLine).toEqual({ lineItem: line, packedSoFar: 1 })

    act(() => result.current.openAddItem())
    expect(result.current.packLine).toBeNull()
    expect(result.current.addItemOpen).toBe(true)
  })

  it('a staged change closes the panel on success and keeps it on failure', async () => {
    vi.mocked(packingApi.refundPackLine).mockRejectedValueOnce(new Error('refund failed')).mockResolvedValueOnce({ scan })
    const { result, opts } = setup()
    act(() => result.current.openLine(line, 0))

    await act(async () => { result.current.confirmRefund(1, '') })
    expect(opts.setError).toHaveBeenLastCalledWith('refund failed')
    expect(result.current.packLine).not.toBeNull()

    await act(async () => { result.current.confirmRefund(1, 'broken') })
    expect(packingApi.refundPackLine).toHaveBeenLastCalledWith('t', 'pack-1', { line_item_id: 'li-1', quantity: 1, reason: 'broken' })
    expect(result.current.packLine).toBeNull()
    expect(opts.loadSession).toHaveBeenCalledWith('pack-1')
  })

  it('substituting resolves each replacement code first', async () => {
    vi.mocked(packingApi.resolvePackScanCode).mockResolvedValue({ sku_id: 'sku-sub', product_name: 'Other' })
    vi.mocked(packingApi.substitutePackLine).mockResolvedValue({ scans: [] })
    const { result } = setup()
    act(() => result.current.openLine(line, 0))
    await act(async () => { result.current.confirmSubstitute([{ code: 'SUB', substituteQuantity: 2, coveredQuantity: 1, returnToStock: true }]) })
    expect(packingApi.resolvePackScanCode).toHaveBeenCalledWith('t', 'pack-1', 'SUB')
    expect(packingApi.substitutePackLine).toHaveBeenCalledWith('t', 'pack-1', {
      line_item_id: 'li-1', lines: [{ sku_id: 'sku-sub', substitute_quantity: 2, original_quantity_covered: 1, return_to_stock: true }],
    })
  })

  it('adding an item stages it as a freebie or a new item', async () => {
    vi.mocked(packingApi.scanPackItem).mockResolvedValue({ scan })
    const { result } = setup()
    await act(async () => { result.current.confirmAddItem({ skuId: 'sku-9', name: 'Jam', quantity: 1, freebie: false, priceMinorUnits: 250 }) })
    expect(packingApi.scanPackItem).toHaveBeenCalledWith('t', 'pack-1', { sku_id: 'sku-9', quantity: 1, action_type: 'new_item', price_minor_units: 250, reason: 'Jam' })
  })
})
