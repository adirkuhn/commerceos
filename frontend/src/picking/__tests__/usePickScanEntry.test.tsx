// @vitest-environment jsdom
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { usePickScanEntry } from '../usePickScanEntry'
import type { PickScan, PickingSessionDetail } from '../../scanner/types'
import type { PickLineSheet } from '../usePickLineSheet'

type Sheet = Pick<PickLineSheet, 'pickLine' | 'pickQty' | 'close'>

vi.mock('../../scanner/errorSound', () => ({ playErrorSound: vi.fn() }))
vi.mock('../../scanner/api', () => ({
  resolvePickScanCode: vi.fn(),
  scanPickItem: vi.fn(),
  deletePickScan: vi.fn(),
}))

import * as pickingApi from '../../scanner/api'
import { playErrorSound } from '../../scanner/errorSound'

const pickScan: PickScan = { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 2, scanned_at: '', scanned_by: 'me' }

function detail(scans: PickScan[] = []): PickingSessionDetail {
  return {
    session: { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'W', barcode: '111', quantity: 2, price_minor_units: 0, total_minor_units: 0 }],
    scans,
  }
}

function setup(session: PickingSessionDetail, sheet: Sheet = { pickLine: null, pickQty: 1, close: vi.fn() }) {
  const loadSession = vi.fn().mockResolvedValue(undefined)
  const setError = vi.fn()
  const hook = renderHook(() => usePickScanEntry({ token: 'tok', session, loadSession, setError, sheet }))
  return { ...hook, loadSession, setError, sheet }
}

async function scanCode(result: ReturnType<typeof setup>['result'], code: string) {
  act(() => result.current.setCode(code))
  await act(() => result.current.handleScanSubmit())
}

beforeEach(() => vi.clearAllMocks())

describe('usePickScanEntry', () => {
  it('picks one unit of a matching line and reloads', async () => {
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue({ scan: pickScan })
    const { result, loadSession } = setup(detail())

    await scanCode(result, ' 111 ')

    expect(pickingApi.resolvePickScanCode).toHaveBeenCalledWith('tok', 'ps-1', '111')
    expect(pickingApi.scanPickItem).toHaveBeenCalledWith('tok', 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 1 })
    expect(loadSession).toHaveBeenCalledWith('ps-1')
    expect(result.current.code).toBe('')
    expect(result.current.scanErrorAlert).toBeNull()
  })

  it('blocks an over-scan of a fully picked line', async () => {
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    const { result } = setup(detail([pickScan]))

    await scanCode(result, '111')

    expect(result.current.scanErrorAlert).toBe('Already fully picked')
    expect(pickingApi.scanPickItem).not.toHaveBeenCalled()
    expect(playErrorSound).toHaveBeenCalled()
  })

  it('blocks a scan the server rejects as not on this order', async () => {
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-x', product_name: 'Gizmo' })
    vi.mocked(pickingApi.scanPickItem).mockRejectedValue(new Error('no'))
    const { result } = setup(detail())

    await scanCode(result, '999')

    expect(result.current.scanErrorAlert).toBe('Item not on this order')
  })

  it('blocks an unknown barcode, ignores further scans until dismissed', async () => {
    vi.mocked(pickingApi.resolvePickScanCode).mockRejectedValue(new Error('not found'))
    const { result } = setup(detail())

    await scanCode(result, 'zzz')
    expect(result.current.scanErrorAlert).toBe('Barcode not recognized')

    await scanCode(result, 'again')
    expect(pickingApi.resolvePickScanCode).toHaveBeenCalledTimes(1)

    act(() => result.current.dismissScanError())
    expect(result.current.scanErrorAlert).toBeNull()
  })

  it('confirms the sheet quantity, closes the sheet and reloads', async () => {
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue({ scan: pickScan })
    const s = detail()
    const { result, loadSession, sheet } = setup(s, { pickLine: { lineItem: s.line_items[0], pickedSoFar: 0 }, pickQty: 2, close: vi.fn() })

    await act(() => result.current.confirmPick())

    expect(pickingApi.scanPickItem).toHaveBeenCalledWith('tok', 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 2 })
    expect(sheet.close).toHaveBeenCalled()
    expect(loadSession).toHaveBeenCalledWith('ps-1')
  })

  it('reports a failed scan removal', async () => {
    vi.mocked(pickingApi.deletePickScan).mockRejectedValue(new Error('gone'))
    const { result, setError, loadSession } = setup(detail([pickScan]))

    await act(() => result.current.handleDeleteScan('scan-1'))

    expect(setError).toHaveBeenLastCalledWith('gone')
    expect(loadSession).not.toHaveBeenCalled()
  })
})
