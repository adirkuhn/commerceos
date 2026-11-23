// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as api from '../api'
import { CodedApiError } from '../../api/codedApiError'
import { POLL_INTERVAL_MS } from '../format'
import { useAssistantRequest } from '../hooks/useAssistantRequest'
import { useAssistantRequests } from '../hooks/useAssistantRequests'
import { useAssistantActions } from '../hooks/useAssistantActions'
import { aDetail, aRequest } from './fixtures'

vi.mock('../api')

beforeEach(() => { vi.mocked(api.getAssistantRequest).mockReset(); vi.mocked(api.listAssistantRequests).mockReset() })
afterEach(() => { vi.useRealTimers() })

describe('useAssistantRequest', () => {
  it('polls while queued or processing and stops once answered', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.mocked(api.getAssistantRequest)
      .mockResolvedValueOnce(aDetail({ status: 'queued' }))
      .mockResolvedValueOnce(aDetail({ status: 'processing' }))
      .mockResolvedValue(aDetail({ status: 'answered' }))

    const { result } = renderHook(() => useAssistantRequest('tok', 'r-1'))
    await waitFor(() => expect(result.current.data?.request.status).toBe('queued'))

    await act(async () => { await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS) })
    expect(result.current.data?.request.status).toBe('processing')
    await act(async () => { await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS) })
    expect(result.current.data?.request.status).toBe('answered')

    await act(async () => { await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3) })
    expect(api.getAssistantRequest).toHaveBeenCalledTimes(3)
  })

  it('does not poll a request waiting for the user', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'needs_input' }))

    const { result } = renderHook(() => useAssistantRequest('tok', 'r-1'))
    await waitFor(() => expect(result.current.data).not.toBeNull())
    await act(async () => { await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3) })

    expect(api.getAssistantRequest).toHaveBeenCalledTimes(1)
  })

  it('surfaces a load error as translated text', async () => {
    vi.mocked(api.getAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_REQUEST_NOT_FOUND', 'not found'))

    const { result } = renderHook(() => useAssistantRequest('tok', 'r-1'))

    await waitFor(() => expect(result.current.error).toBe('This assistant request doesn’t exist (any more).'))
  })

  it('setData replaces what is shown and restarts polling for a re-queued request', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.mocked(api.getAssistantRequest).mockResolvedValueOnce(aDetail({ status: 'answered' })).mockResolvedValue(aDetail({ status: 'answered' }))

    const { result } = renderHook(() => useAssistantRequest('tok', 'r-1'))
    await waitFor(() => expect(result.current.data?.request.status).toBe('answered'))

    act(() => { result.current.setRequest(aRequest({ status: 'queued' })) })
    expect(result.current.data?.request.status).toBe('queued')
    await act(async () => { await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS) })
    expect(api.getAssistantRequest).toHaveBeenCalledTimes(2)
  })
})

describe('useAssistantRequests', () => {
  it('loads the filtered first page, then appends the next one', async () => {
    vi.mocked(api.listAssistantRequests)
      .mockResolvedValueOnce({ requests: [aRequest({ id: 'r-1' })], next_cursor: 'r-1' })
      .mockResolvedValueOnce({ requests: [aRequest({ id: 'r-2' })], next_cursor: null })

    const { result } = renderHook(() => useAssistantRequests('tok', 'queued'))
    await waitFor(() => expect(result.current.items).toHaveLength(1))
    expect(result.current.hasMore).toBe(true)
    expect(api.listAssistantRequests).toHaveBeenLastCalledWith('tok', { status: 'queued' })

    await act(async () => { result.current.loadMore() })
    await waitFor(() => expect(result.current.items.map(r => r.id)).toEqual(['r-1', 'r-2']))
    expect(api.listAssistantRequests).toHaveBeenLastCalledWith('tok', { status: 'queued', cursor: 'r-1' })
    expect(result.current.hasMore).toBe(false)
  })

  it('starts over when the filter changes', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({ requests: [aRequest()], next_cursor: null })

    const { result, rerender } = renderHook(({ s }) => useAssistantRequests('tok', s), { initialProps: { s: undefined as 'answered' | undefined } })
    await waitFor(() => expect(result.current.items).toHaveLength(1))
    rerender({ s: 'answered' })

    await waitFor(() => expect(api.listAssistantRequests).toHaveBeenLastCalledWith('tok', { status: 'answered' }))
    await waitFor(() => expect(result.current.items).toHaveLength(1))
  })

  it('replace swaps one row in place (after a cancel)', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({ requests: [aRequest()], next_cursor: null })
    const { result } = renderHook(() => useAssistantRequests('tok', undefined))
    await waitFor(() => expect(result.current.items).toHaveLength(1))

    act(() => { result.current.replace(aRequest({ status: 'cancelled' })) })

    expect(result.current.items[0].status).toBe('cancelled')
  })

  it('upsert puts a new chat on top and updates an existing one in place', async () => {
    vi.mocked(api.listAssistantRequests).mockResolvedValue({ requests: [aRequest({ id: 'r-1' })], next_cursor: null })
    const { result } = renderHook(() => useAssistantRequests('tok', undefined))
    await waitFor(() => expect(result.current.items).toHaveLength(1))

    act(() => { result.current.upsert(aRequest({ id: 'r-2', title: 'New' })) })
    act(() => { result.current.upsert(aRequest({ id: 'r-1', status: 'answered' })) })

    expect(result.current.items.map(r => r.id)).toEqual(['r-2', 'r-1'])
    expect(result.current.items[1].status).toBe('answered')
  })

  it('surfaces a list error', async () => {
    vi.mocked(api.listAssistantRequests).mockRejectedValue(new Error('network down'))
    const { result } = renderHook(() => useAssistantRequests('tok', undefined))
    await waitFor(() => expect(result.current.error).toBe('network down'))
  })
})

describe('useAssistantActions', () => {
  it('creates, replies and cancels, returning the request', async () => {
    vi.mocked(api.createAssistantRequest).mockResolvedValue(aRequest({ id: 'r-new' }))
    vi.mocked(api.replyToAssistantRequest).mockResolvedValue(aRequest({ status: 'queued' }))
    vi.mocked(api.cancelAssistantRequest).mockResolvedValue(aRequest({ status: 'cancelled' }))
    const { result } = renderHook(() => useAssistantActions('tok'))

    let created: unknown
    await act(async () => { created = await result.current.create('hi') })
    expect(created).toMatchObject({ id: 'r-new' })
    await act(async () => { await result.current.reply('r-1', 'Ana Bakery') })
    expect(api.replyToAssistantRequest).toHaveBeenCalledWith('tok', 'r-1', 'Ana Bakery')
    await act(async () => { await result.current.cancel('r-1') })
    expect(api.cancelAssistantRequest).toHaveBeenCalledWith('tok', 'r-1')
    expect(result.current.error).toBeNull()
  })

  it('returns null and keeps the translated error on failure', async () => {
    vi.mocked(api.cancelAssistantRequest).mockRejectedValue(new CodedApiError('ASSISTANT_NOT_CANCELLABLE', 'x'))
    const { result } = renderHook(() => useAssistantActions('tok'))

    let out: unknown = 'unset'
    await act(async () => { out = await result.current.cancel('r-1') })

    expect(out).toBeNull()
    expect(result.current.error).toBe('Only a request that hasn’t finished can be cancelled.')
    expect(result.current.busy).toBe(false)
  })

  it('retries a waiting request now', async () => {
    vi.mocked(api.retryAssistantRequest).mockResolvedValue(aRequest({ next_attempt_at: '2026-11-23T12:00:00Z' }))
    const { result } = renderHook(() => useAssistantActions('tok'))

    let out: unknown
    await act(async () => { out = await result.current.retry('r-1') })

    expect(api.retryAssistantRequest).toHaveBeenCalledWith('tok', 'r-1')
    expect(out).toMatchObject({ id: 'r-1' })
  })
})
