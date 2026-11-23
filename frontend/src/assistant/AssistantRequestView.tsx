import { ArrowLeft, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '../components/common/useConfirm'
import { useAuth } from '../auth/AuthContext'
import { useStackNavigation } from '../routing/useStackNavigation'
import { Conversation } from './Conversation'
import { ASSISTANT_PATH, canReply, isCancellable, isWorking } from './format'
import { useAssistantActions } from './hooks/useAssistantActions'
import { useAssistantRequest } from './hooks/useAssistantRequest'
import { useProposalActions } from './hooks/useProposalActions'
import { ProposalCard } from './ProposalCard'
import { ReplyBox, type ReplyMode } from './ReplyBox'
import type { AssistantRequestStatus } from './types'
import { StatusChip } from './StatusChip'
import { useCancelRequest } from './useCancelRequest'
import { WaitReason } from './WaitReason'

// /assistant/<id>: one request's conversation, refreshed while the model
// works on it; reply or cancel from here.
export function AssistantRequestView({ id }: { id: string }) {
  const { t } = useTranslation(['assistant', 'common'])
  const { token } = useAuth()
  const { exitToOpener } = useStackNavigation()
  const detail = useAssistantRequest(token ?? '', id)
  const actions = useAssistantActions(token ?? '')
  const { cancel, confirmDialog } = useCancelRequest(actions, detail.setRequest)
  const proposals = useProposalActions(token ?? '', id, detail.setDetail)
  const { confirm: ask, confirmDialog: rejectDialog } = useConfirm()

  async function reject(proposalId: string) {
    const ok = await ask({ title: t('proposal.rejectTitle'), message: t('proposal.rejectMessage'), confirmLabel: t('proposal.rejectConfirm'), destructive: true })
    if (ok) await proposals.reject(proposalId)
  }

  function renderProposal(proposalId: string) {
    const p = detail.data?.proposals.find(x => x.id === proposalId)
    if (!p) return null
    return (
      <ProposalCard
        key={p.id} proposal={p} actionable={detail.data?.request.status === 'awaiting_confirmation'}
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

  const request = detail.data?.request
  return (
    <div className="p-5 flex flex-col gap-4 max-w-3xl">
      <button type="button" onClick={() => exitToOpener(ASSISTANT_PATH)} className="self-start flex items-center gap-1 text-xs text-ink-3 hover:text-ink">
        <ArrowLeft size={12} aria-hidden /> {t('conversation.back')}
      </button>

      {detail.error && <p className="text-xs text-danger">{detail.error}</p>}
      {detail.loading && !detail.data && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}

      {request && detail.data && (
        <>
          <header className="flex items-start gap-3">
            <h2 className="flex-1 min-w-0 text-sm font-semibold text-ink break-words">{request.title}</h2>
            <StatusChip status={request.status} />
            {isCancellable(request.status) && (
              <button type="button" onClick={() => cancel(id)} className="text-xs text-ink-3 hover:text-danger whitespace-nowrap">
                {t('cancel.action')}
              </button>
            )}
          </header>

          <Conversation messages={detail.data.messages} renderProposal={renderProposal} />

          {isWorking(request.status) && (
            <div className="flex flex-col gap-1 rounded-lg border border-line bg-sunken px-3 py-2">
              <p className="flex items-center gap-1.5 text-xs text-ink-3">
                <Loader2 size={12} className="animate-spin" aria-hidden />
                {request.status === 'processing' ? t('conversation.working') : t('conversation.queued')}
              </p>
              <WaitReason request={request} detailed />
            </div>
          )}

          {actions.error && <p className="text-xs text-danger">{actions.error}</p>}
          {canReply(request.status) && <ReplyBox mode={replyMode(request.status)} busy={actions.busy} onSend={send} />}
        </>
      )}
      {confirmDialog}
      {rejectDialog}
    </div>
  )
}

function replyMode(status: AssistantRequestStatus): ReplyMode {
  if (status === 'answered') return 'followUp'
  if (status === 'awaiting_confirmation') return 'changes'
  return 'answer'
}
