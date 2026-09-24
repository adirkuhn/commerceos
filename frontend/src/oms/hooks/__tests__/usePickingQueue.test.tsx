// @vitest-environment jsdom
import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { usePickingQueue } from '../usePackingQueue'

vi.mock('../../api', () => ({ listPickingQueue: vi.fn(), listPackingQueue: vi.fn() }))
import * as api from '../../api'

describe('usePickingQueue', () => {
  it('loads the picking queue', async () => {
    const entry = { sales_order: { id: 'so-1' }, lock: null, in_packing: true } as Awaited<ReturnType<typeof api.listPickingQueue>>['entries'][number]
    vi.mocked(api.listPickingQueue).mockResolvedValue({ entries: [entry], total: 1 })

    const { result } = renderHook(() => usePickingQueue('tok'))

    await waitFor(() => expect(result.current.entries).toEqual([entry]))
    expect(api.listPickingQueue).toHaveBeenCalledWith('tok', { perPage: 100 })
    expect(api.listPackingQueue).not.toHaveBeenCalled()
  })
})
