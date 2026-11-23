import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import { useConfirm } from '../../components/common/useConfirm'
import { canReply, isCancellable, isWorking } from '../format'
import { useAssistantActions } from '../hooks/useAssistantActions'
import { useAssistantRequest } from '../hooks/useAssistantRequest'
import { useProposalActions } from '../hooks/useProposalActions'
import { ProposalCard } from '../ProposalCard'
import { StatusChip } from '../StatusChip'
import type { AssistantRequest, AssistantRequestStatus } from '../types'
import { useCancelRequest } from '../useCancelRequest'
import { Composer, type ComposerMode } from './Composer'
import { Thread } from './Thread'
import { WaitingStatus } from './WaitingStatus'

interface Props {
  id: string
  // Keeps the chat history in step with what this chat shows.
  onRequest: (request: AssistantRequest) => void
  // Bumped when the history changed this chat (cancel, retry): re-read it.
  refreshSignal: number
}

// One chat: the thread, the assistant's state while it works, and the
// composer pinned at the bottom.
export function ChatThread({ id, onRequest, refreshSignal }: Props) {
  const { t } = useTranslation(['assistant', 'common'])
  const { token } = useAuth()
  const detail = useAssistantRequest(token ?? '', id)
  const actions = useAssistantActions(token ?? '')
  const { cancel, confirmDialog } = useCancelRequest(actions, detail.setRequest)
  const proposals = useProposalActions(token ?? '', id, detail.setDetail)
  const { confirm: ask, confirmDialog: rejectDialog } = useConfirm()
  const bottom = useRef<HTMLDivElement>(null)
  const firstSignal = useRef(refreshSignal)

  const request = detail.data?.request
  useEffect(() => { if (request) onRequest(request) }, [request, onRequest])
  useEffect(() => {
    if (refreshSignal !== firstSignal.current) detail.refresh()
  }, [refreshSignal]) // eslint-disable-line react-hooks/exhaustive-deps -- only on a new signal
  const messageCount = detail.data?.messages.length ?? 0
  useEffect(() => { bottom.current?.scrollIntoView?.({ block: 'end' }) }, [messageCount, request?.status])

  async function reject(proposalId: string) {
    const ok = await ask({ title: t('proposal.rejectTitle'), message: t('proposal.rejectMessage'), confirmLabel: t('proposal.rejectConfirm'), destructive: true })
    if (ok) await proposals.reject(proposalId)
  }

  function renderProposal(proposalId: string) {
    const p = detail.data?.proposals.find(x => x.id === proposalId)
    if (!p) return null
    return (
      <ProposalCard
        key={p.id} proposal={p} actionable={request?.status === 'awaiting_confirmation'}
        busy={proposals.busyId === p.id} error={proposals.errors[p.id]}
        onConfirm={warehouseId => proposals.confirm(p.id, warehouseId)} onReject={() => reject(p.id)}
      />
    )
  }

  async function send(text: string): Promise<boolean> {
    const out = await actions.reply(id, text)
    if (!out) return false
    detail.setRequest(out)
    return true
  }

  async function retry() {
    const out = await actions.retry(id)
    if (!out) return
    detail.setRequest(out)
    detail.refresh()
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      {request && (
        <header className="flex items-center gap-3 px-5 py-2.5 border-b border-line shrink-0">
          <h2 className="flex-1 min-w-0 text-sm font-semibold text-ink truncate">{request.title}</h2>
          <StatusChip status={request.status} />
          {isCancellable(request.status) && (
            <button type="button" onClick={() => cancel(id)} className="text-xs text-ink-3 hover:text-danger whitespace-nowrap">
              {t('cancel.action')}
            </button>
          )}
        </header>
      )}

      <div className="flex-1 min-h-0 overflow-auto px-5 py-4">
        <div className="max-w-3xl mx-auto flex flex-col gap-3">
          {detail.error && <p className="text-xs text-danger">{detail.error}</p>}
          {detail.loading && !detail.data && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
          {detail.data && request && (
            <Thread messages={detail.data.messages} proposals={detail.data.proposals} renderProposal={renderProposal}>
              <WaitingStatus request={request} retrying={actions.busy} onRetry={retry} />
            </Thread>
          )}
          <div ref={bottom} />
        </div>
      </div>

      {request && (
        <div className="shrink-0 border-t border-line px-5 py-3">
          <div className="max-w-3xl mx-auto flex flex-col gap-1.5">
            {actions.error && <p className="text-xs text-danger">{actions.error}</p>}
            {request.status === 'cancelled'
              ? <p className="text-xs text-ink-4">{t('conversation.cancelledHint')}</p>
              : (
                <Composer
                  mode={composerMode(request.status)} busy={actions.busy} onSend={send}
                  disabled={!canReply(request.status)} hint={isWorking(request.status) ? t('conversation.waitHint') : undefined}
                />
              )}
          </div>
        </div>
      )}
      {confirmDialog}
      {rejectDialog}
    </div>
  )
}

function composerMode(status: AssistantRequestStatus): ComposerMode {
  if (status === 'needs_input') return 'answer'
  if (status === 'awaiting_confirmation') return 'changes'
  return 'followUp'
}
