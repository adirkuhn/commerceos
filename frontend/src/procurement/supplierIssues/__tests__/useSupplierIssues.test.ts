// @vitest-environment jsdom
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../../api', () => ({ listSupplierIssues: vi.fn(), updateSupplierIssueStatus: vi.fn(), emailSupplierIssues: vi.fn() }))

import * as api from '../../api'
import { useSupplierIssues } from '../useSupplierIssues'
import type { SupplierIssue, SupplierIssueList } from '../../types'
import { CodedApiError } from '../../../api/codedApiError'

const issue: SupplierIssue = {
  source: 'booking_scan', source_id: 'scan-1', kind: 'damaged', supplier_sku: '', supplier_description: 'Guarana 2L',
  quantity: 2, reason: '', reported_by: '', occurred_at: '2026-10-01T12:00:00Z', status: 'open', status_note: '',
}
const list: SupplierIssueList = { purchase_order_id: 'po-1', booking_in_progress: false, issues: [issue] }

beforeEach(() => vi.clearAllMocks())

describe('useSupplierIssues', () => {
  it('loads the purchase order’s issues', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(list)

    const { result } = renderHook(() => useSupplierIssues('tok', 'po-1'))

    await waitFor(() => expect(result.current.list).toEqual(list))
    expect(api.listSupplierIssues).toHaveBeenCalledWith('tok', 'po-1')
    expect(result.current.error).toBeNull()
  })

  it('shows a load failure', async () => {
    vi.mocked(api.listSupplierIssues).mockRejectedValue(new Error('offline'))

    const { result } = renderHook(() => useSupplierIssues('tok', 'po-1'))

    await waitFor(() => expect(result.current.error).toBe('offline'))
    expect(result.current.list).toBeNull()
  })

  it('swaps in the updated issue after a status change', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(list)
    vi.mocked(api.updateSupplierIssueStatus).mockResolvedValue({ ...issue, status: 'resolved', status_note: 'CN-7' })
    const { result } = renderHook(() => useSupplierIssues('tok', 'po-1'))
    await waitFor(() => expect(result.current.list).not.toBeNull())

    await act(() => result.current.updateStatus(issue, 'resolved', 'CN-7'))

    expect(api.updateSupplierIssueStatus).toHaveBeenCalledWith('tok', 'po-1', issue, 'resolved', 'CN-7')
    expect(result.current.list?.issues[0]).toMatchObject({ status: 'resolved', status_note: 'CN-7' })
  })

  it('rethrows a failed status change for the caller to show', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(list)
    vi.mocked(api.updateSupplierIssueStatus).mockRejectedValue(new CodedApiError('SUPPLIER_ISSUE_NOT_FOUND', 'gone'))
    const { result } = renderHook(() => useSupplierIssues('tok', 'po-1'))
    await waitFor(() => expect(result.current.list).not.toBeNull())

    await expect(result.current.updateStatus(issue, 'claimed', '')).rejects.toThrow()
    expect(result.current.list?.issues[0].status).toBe('open')
  })

  it('reloads after emailing, so claimed statuses show', async () => {
    vi.mocked(api.listSupplierIssues).mockResolvedValue(list)
    vi.mocked(api.emailSupplierIssues).mockResolvedValue({ sent_to: 'a@b.example', claimed: 1 })
    const { result } = renderHook(() => useSupplierIssues('tok', 'po-1'))
    await waitFor(() => expect(result.current.list).not.toBeNull())
    const input = { to: 'a@b.example', subject: 's', body: 'b', issues: [{ source: issue.source, source_id: issue.source_id }] }

    let sent
    await act(async () => { sent = await result.current.email(input) })

    expect(sent).toEqual({ sent_to: 'a@b.example', claimed: 1 })
    expect(api.emailSupplierIssues).toHaveBeenCalledWith('tok', 'po-1', input)
    expect(api.listSupplierIssues).toHaveBeenCalledTimes(2)
  })
})
