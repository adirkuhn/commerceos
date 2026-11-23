import i18next from 'i18next'
import { useCallback, useState } from 'react'
import { apiErrorText } from '../../api/apiErrorText'
import { confirmAssistantProposal, rejectAssistantProposal } from '../api'
import type { AssistantRequestDetail } from '../types'

export interface ProposalActions {
  confirm: (proposalId: string, warehouseId?: string) => Promise<void>
  reject: (proposalId: string) => Promise<void>
  // The proposal a call is running for -- its buttons are disabled, so a
  // double click can't decide it twice.
  busyId: string | null
  // Why the last call failed, per proposal (shown in that card).
  errors: Record<string, string>
}

// Confirm or reject one of the request's proposals. Both answer with the
// whole updated request, handed to onDetail.
export function useProposalActions(token: string, requestId: string, onDetail: (d: AssistantRequestDetail) => void): ProposalActions {
  const [busyId, setBusyId] = useState<string | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const run = useCallback(async (proposalId: string, call: () => Promise<AssistantRequestDetail>) => {
    setBusyId(proposalId)
    setErrors(({ [proposalId]: _dropped, ...rest }) => rest)
    try {
      onDetail(await call())
    } catch (e) {
      setErrors(prev => ({ ...prev, [proposalId]: apiErrorText(e) ?? i18next.t('ui:errors.unknown') }))
    } finally {
      setBusyId(null)
    }
  }, [onDetail])

  const confirm = useCallback((proposalId: string, warehouseId?: string) =>
    run(proposalId, () => confirmAssistantProposal(token, requestId, proposalId, warehouseId ? { warehouse_id: warehouseId } : {})),
  [run, token, requestId])
  const reject = useCallback((proposalId: string) =>
    run(proposalId, () => rejectAssistantProposal(token, requestId, proposalId)),
  [run, token, requestId])

  return { confirm, reject, busyId, errors }
}
