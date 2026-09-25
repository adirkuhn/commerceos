// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { createRef, type FormEvent } from 'react'
import { usePickingScan } from '../usePickingScan'
import type { PickingSessionDetail } from '../types'
import type { SKU } from '../../pms/types'
import type { SKUCandidate } from '../../pms/searchCandidates'

vi.mock('../api', () => ({ resolvePickScanCode: vi.fn() }))
vi.mock('../../pms/searchCandidates', () => ({ searchSKUCandidates: vi.fn() }))
vi.mock('../errorSound', () => ({ playErrorSound: vi.fn() }))

import * as api from '../api'
import { searchSKUCandidates } from '../../pms/searchCandidates'

const detail: PickingSessionDetail = {
  session: { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' },
  line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'W', barcode: '111', quantity: 1, price_minor_units: 0, total_minor_units: 0 }],
  scans: [],
}
const candidate = (id: string, pricingMode: SKU['pricing_mode'] = 'fixed'): SKUCandidate => ({ sku: { id, pricing_mode: pricingMode } as SKU, productId: 'p', productName: 'P' })
const submit = { preventDefault: vi.fn() } as unknown as FormEvent

let stage: ReturnType<typeof vi.fn>
let setError: ReturnType<typeof vi.fn>
let setScanning: ReturnType<typeof vi.fn>
let onAuthError: ReturnType<typeof vi.fn>

function setup(d: PickingSessionDetail = detail) {
  return renderHook(() => usePickingScan({
    token: 't', sessionId: 'ps-1', detail: d, codeInputRef: createRef<HTMLInputElement>(),
    stage, setError, setScanning, onAuthError,
  }))
}

async function typeAndScan(result: ReturnType<typeof setup>['result'], code: string) {
  act(() => result.current.setCode(code))
  await act(async () => { result.current.handleScan(submit) })
}

beforeEach(() => {
  vi.clearAllMocks()
  stage = vi.fn().mockResolvedValue(true)
  setError = vi.fn()
  setScanning = vi.fn()
  onAuthError = vi.fn()
})

describe('usePickingScan', () => {
  it('ignores a blank code', async () => {
    const { result } = setup()
    await typeAndScan(result, '   ')
    expect(api.resolvePickScanCode).not.toHaveBeenCalled()
  })

  it('resolves the trimmed code and stages one unit, clearing the box', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    const { result } = setup()
    await typeAndScan(result, ' 111 ')

    expect(api.resolvePickScanCode).toHaveBeenCalledWith('t', 'ps-1', '111')
    await waitFor(() => expect(stage).toHaveBeenCalledWith({ sku_id: 'sku-1', action_type: 'pick', quantity: 1 }))
    expect(result.current.code).toBe('')
    expect(setScanning).toHaveBeenLastCalledWith(false)
  })

  it('refuses an over-scan without staging', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    const { result } = setup({ ...detail, scans: [{ id: 's', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '', scanned_by: '' }] })
    await typeAndScan(result, '111')

    await waitFor(() => expect(result.current.scanErrorAlert).toBe('Already fully picked'))
    expect(stage).not.toHaveBeenCalled()
  })

  it('a SKU off the order that the server refuses becomes a blocking error, not an inline one', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-x', product_name: 'X', weight_grams: 500 })
    stage.mockResolvedValue(false)
    const { result } = setup()
    await typeAndScan(result, '222')

    await waitFor(() => expect(result.current.scanErrorAlert).toBe('Item not on this order'))
    expect(stage).toHaveBeenCalledWith({ sku_id: 'sku-x', action_type: 'pick', quantity: 0.5 })
    expect(setError).toHaveBeenLastCalledWith('')
  })

  it('no further scans are resolved while an error is showing', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('nope'))
    vi.mocked(searchSKUCandidates).mockResolvedValue([])
    const { result } = setup()
    await typeAndScan(result, 'zzz')
    expect(result.current.scanErrorAlert).toBe('Barcode not recognized')

    await typeAndScan(result, 'again')
    expect(api.resolvePickScanCode).toHaveBeenCalledTimes(1)

    act(() => result.current.dismissScanError())
    expect(result.current.scanErrorAlert).toBeNull()
  })

  it('a failing search is treated as no matches', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('nope'))
    vi.mocked(searchSKUCandidates).mockRejectedValue(new Error('search down'))
    const { result } = setup()
    await typeAndScan(result, 'zzz')
    expect(result.current.scanErrorAlert).toBe('Barcode not recognized')
  })

  it('an auth failure hands off without searching', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('401'))
    const { result } = setup()
    await typeAndScan(result, 'zzz')
    expect(onAuthError).toHaveBeenCalled()
    expect(searchSKUCandidates).not.toHaveBeenCalled()
  })

  it('search matches become candidates; choosing one routes it', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('nope'))
    vi.mocked(searchSKUCandidates).mockResolvedValue([candidate('sku-1')])
    const { result } = setup()
    await typeAndScan(result, 'wid')
    expect(result.current.candidates).toHaveLength(1)
    expect(result.current.code).toBe('')

    await act(async () => { result.current.selectCandidate(result.current.candidates[0]) })

    expect(result.current.candidates).toEqual([])
    expect(stage).toHaveBeenCalledWith({ sku_id: 'sku-1', action_type: 'pick', quantity: 1 })
  })

  it('a by-weight candidate is refused', async () => {
    const { result } = setup()
    act(() => result.current.selectCandidate(candidate('sku-w', 'per_weight')))
    expect(result.current.scanErrorAlert).toMatch(/Priced by weight/)
    expect(stage).not.toHaveBeenCalled()
  })

  it('a camera decode closes the camera and resolves the code', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    const { result } = setup()
    act(() => result.current.setCameraOpen(true))

    await act(async () => { result.current.handleCameraDecode('111') })

    expect(result.current.cameraOpen).toBe(false)
    expect(api.resolvePickScanCode).toHaveBeenCalledWith('t', 'ps-1', '111')
  })
})
