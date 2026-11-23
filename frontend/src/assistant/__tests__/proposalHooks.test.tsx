// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import * as api from '../api'
import { CodedApiError } from '../../api/codedApiError'
import { useAssistantRequest } from '../hooks/useAssistantRequest'
import { useProposalActions } from '../hooks/useProposalActions'
import { aDetail, aProposal } from './fixtures'

vi.mock('../api')

beforeEach(() => { vi.mocked(api.getAssistantRequest).mockReset() })

describe('useProposalActions', () => {
  it('confirms with the chosen warehouse and hands back the updated detail', async () => {
    const after = aDetail({ status: 'answered' }, [], [aProposal({ status: 'confirmed', result_ref: 'Q-0012' })])
    vi.mocked(api.confirmAssistantProposal).mockResolvedValue(after)
    const onDetail = vi.fn()
    const { result } = renderHook(() => useProposalActions('tok', 'r-1', onDetail))

    await act(async () => { await result.current.confirm('p-1', 'w-2') })

    expect(api.confirmAssistantProposal).toHaveBeenCalledWith('tok', 'r-1', 'p-1', { warehouse_id: 'w-2' })
    expect(onDetail).toHaveBeenCalledWith(after)
    expect(result.current.busyId).toBeNull()
  })

  it('confirms without a warehouse for a quote', async () => {
    vi.mocked(api.confirmAssistantProposal).mockResolvedValue(aDetail())
    const { result } = renderHook(() => useProposalActions('tok', 'r-1', vi.fn()))

    await act(async () => { await result.current.confirm('p-1') })

    expect(api.confirmAssistantProposal).toHaveBeenCalledWith('tok', 'r-1', 'p-1', {})
  })

  it('rejects', async () => {
    vi.mocked(api.rejectAssistantProposal).mockResolvedValue(aDetail())
    const onDetail = vi.fn()
    const { result } = renderHook(() => useProposalActions('tok', 'r-1', onDetail))

    await act(async () => { await result.current.reject('p-1') })

    expect(api.rejectAssistantProposal).toHaveBeenCalledWith('tok', 'r-1', 'p-1')
    expect(onDetail).toHaveBeenCalled()
  })

  it('keeps the translated error per proposal and marks the busy one while running', async () => {
    let fail: (e: unknown) => void = () => {}
    vi.mocked(api.confirmAssistantProposal).mockReturnValue(new Promise((_, reject) => { fail = reject }))
    const onDetail = vi.fn()
    const { result } = renderHook(() => useProposalActions('tok', 'r-1', onDetail))

    let done: Promise<void> = Promise.resolve()
    act(() => { done = result.current.confirm('p-1') })
    expect(result.current.busyId).toBe('p-1')
    await act(async () => { fail(new CodedApiError('ASSISTANT_PROPOSAL_NOT_PENDING', 'x')); await done })

    expect(result.current.busyId).toBeNull()
    expect(result.current.errors['p-1']).toBe('This proposal was already decided. Refresh to see what happened.')
    expect(onDetail).not.toHaveBeenCalled()
  })
})

describe('useAssistantRequest.setDetail', () => {
  it('replaces the whole detail (after a decision)', async () => {
    vi.mocked(api.getAssistantRequest).mockResolvedValue(aDetail({ status: 'awaiting_confirmation' }))
    const { result } = renderHook(() => useAssistantRequest('tok', 'r-1'))
    await waitFor(() => expect(result.current.data).not.toBeNull())

    act(() => { result.current.setDetail(aDetail({ status: 'answered' }, [], [aProposal({ status: 'rejected' })])) })

    expect(result.current.data?.request.status).toBe('answered')
    expect(result.current.data?.proposals[0].status).toBe('rejected')
  })
})
