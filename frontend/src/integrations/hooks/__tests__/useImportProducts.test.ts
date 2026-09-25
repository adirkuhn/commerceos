// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type React from 'react'

vi.mock('../../api', () => ({ importWooProducts: vi.fn() }))

import * as api from '../../api'
import { useImportProducts } from '../useImportProducts'
import type { Connection, ImportJob } from '../../types'

const event = () => ({ preventDefault: vi.fn() }) as unknown as React.FormEvent
const woo = (overrides: Partial<Connection> = {}): Connection =>
  ({ id: 'c1', provider: 'woocommerce', store_url: '', status: 'connected', created_at: '', ...overrides })

beforeEach(() => vi.clearAllMocks())

describe('useImportProducts', () => {
  it('defaults the currency from the connection, normalized', async () => {
    const { result } = renderHook(() => useImportProducts('tok', woo({ default_currency: ' eur ' }), 'USD'))
    await waitFor(() => expect(result.current.currency).toBe('EUR'))
  })

  it('falls back to the tenant currency', async () => {
    const { result } = renderHook(() => useImportProducts('tok', woo(), 'gbp'))
    await waitFor(() => expect(result.current.currency).toBe('GBP'))
  })

  it('submits and stores the started job', async () => {
    const job = { id: 'j1' } as ImportJob
    vi.mocked(api.importWooProducts).mockResolvedValue(job)
    const { result } = renderHook(() => useImportProducts('tok', woo({ default_currency: 'EUR' }), undefined))
    await waitFor(() => expect(result.current.currency).toBe('EUR'))
    act(() => result.current.setWarehouseId('wh-1'))
    await act(() => result.current.submit(event()))
    expect(api.importWooProducts).toHaveBeenCalledWith('tok', 'c1', 'wh-1', 'EUR')
    expect(result.current.job).toBe(job)
    expect(result.current.importing).toBe(false)
  })

  it('surfaces a failure', async () => {
    vi.mocked(api.importWooProducts).mockRejectedValue(new Error('already running'))
    const { result } = renderHook(() => useImportProducts('tok', woo(), undefined))
    await act(() => result.current.submit(event()))
    expect(result.current.error).toBe('already running')
    expect(result.current.job).toBeNull()
  })
})
