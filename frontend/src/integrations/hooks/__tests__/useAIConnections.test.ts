// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../api', () => ({
  listAIConnections: vi.fn(),
  createAIConnection: vi.fn(),
  updateAIConnection: vi.fn(),
  deleteAIConnection: vi.fn(),
  reorderAIConnections: vi.fn(),
}))

import * as api from '../../api'
import type { AIConnection } from '../../types'
import { useAIConnections } from '../useAIConnections'

function conn(id: string, position: number, extra: Partial<AIConnection> = {}): AIConnection {
  return { id, provider: 'nvidia', position, label: '', enabled: true, has_api_key: true, status: 'connected', created_at: '', ...extra }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listAIConnections).mockResolvedValue([conn('a', 0), conn('b', 1), conn('c', 2)])
})

async function loaded() {
  const hook = renderHook(() => useAIConnections('tok'))
  await waitFor(() => expect(hook.result.current.connections).toHaveLength(3))
  return hook
}

describe('useAIConnections', () => {
  it('loads the list in routing order', async () => {
    const { result } = await loaded()
    expect(result.current.connections.map(c => c.id)).toEqual(['a', 'b', 'c'])
    expect(api.listAIConnections).toHaveBeenCalledWith('tok')
  })

  it('appends a created connection', async () => {
    vi.mocked(api.createAIConnection).mockResolvedValue(conn('d', 3))
    const { result } = await loaded()

    await act(async () => { await result.current.create({ provider: 'gemini', apiKey: 'k' }) })

    expect(api.createAIConnection).toHaveBeenCalledWith('tok', { provider: 'gemini', apiKey: 'k' })
    expect(result.current.connections.map(c => c.id)).toEqual(['a', 'b', 'c', 'd'])
  })

  it('replaces an updated connection in place', async () => {
    vi.mocked(api.updateAIConnection).mockResolvedValue(conn('b', 1, { enabled: false }))
    const { result } = await loaded()

    await act(async () => { await result.current.update('b', { enabled: false }) })

    expect(api.updateAIConnection).toHaveBeenCalledWith('tok', 'b', { enabled: false })
    expect(result.current.connections[1].enabled).toBe(false)
  })

  it('removes and reloads (positions close up server-side)', async () => {
    vi.mocked(api.deleteAIConnection).mockResolvedValue(null)
    const { result } = await loaded()
    vi.mocked(api.listAIConnections).mockResolvedValue([conn('b', 0), conn('c', 1)])

    await act(async () => { await result.current.remove('a') })

    expect(api.deleteAIConnection).toHaveBeenCalledWith('tok', 'a')
    expect(result.current.connections.map(c => [c.id, c.position])).toEqual([['b', 0], ['c', 1]])
  })

  it('reorders optimistically and keeps the server answer', async () => {
    let resolve: (v: AIConnection[]) => void = () => {}
    vi.mocked(api.reorderAIConnections).mockReturnValue(new Promise(r => { resolve = r }))
    const { result } = await loaded()

    let done: Promise<void> = Promise.resolve()
    act(() => { done = result.current.reorder(['c', 'a', 'b']) })
    expect(result.current.connections.map(c => c.id)).toEqual(['c', 'a', 'b'])

    await act(async () => { resolve([conn('c', 0), conn('a', 1), conn('b', 2)]); await done })
    expect(api.reorderAIConnections).toHaveBeenCalledWith('tok', ['c', 'a', 'b'])
    expect(result.current.connections.map(c => c.position)).toEqual([0, 1, 2])
    expect(result.current.error).toBeNull()
  })

  it('rolls a failed reorder back and surfaces the error', async () => {
    vi.mocked(api.reorderAIConnections).mockRejectedValue(new Error('the new order must list every AI connection exactly once'))
    const { result } = await loaded()

    await act(async () => { await result.current.reorder(['c', 'a', 'b']) })

    expect(result.current.connections.map(c => c.id)).toEqual(['a', 'b', 'c'])
    expect(result.current.error).toBe('the new order must list every AI connection exactly once')
  })

  it('surfaces a load error', async () => {
    vi.mocked(api.listAIConnections).mockRejectedValue(new Error('boom'))
    const { result } = renderHook(() => useAIConnections('tok'))

    await waitFor(() => expect(result.current.error).toBe('boom'))
    expect(result.current.loading).toBe(false)
  })
})
