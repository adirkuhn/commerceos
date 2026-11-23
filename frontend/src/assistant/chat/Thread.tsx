import clsx from 'clsx'
import { ChevronRight } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { quotePath, salesOrderPath } from '../format'
import { Markdown } from '../markdown/Markdown'
import type { AssistantMessage, AssistantProposal } from '../types'

type Item =
  | { kind: 'message'; message: AssistantMessage }
  | { kind: 'tools'; key: number; names: string[] }

// Turns the message log into what the chat shows: assistant steps that
// only called tools have no text of their own and are skipped, and the
// tool results between two visible messages fold into one "Looked up N
// things" line. A tool message that made a proposal stays on its own: it
// shows the proposal's card.
export function threadItems(messages: AssistantMessage[]): Item[] {
  const items: Item[] = []
  for (const m of messages) {
    if (m.role === 'tool' && m.notice !== 'proposed') {
      const last = items[items.length - 1]
      if (last?.kind === 'tools') last.names.push(m.tool_name ?? '')
      else items.push({ kind: 'tools', key: m.seq, names: [m.tool_name ?? ''] })
      continue
    }
    if (m.role === 'assistant' && !m.content && !m.notice) continue
    items.push({ kind: 'message', message: m })
  }
  return items
}

interface Props {
  messages: AssistantMessage[]
  proposals: AssistantProposal[]
  renderProposal?: (proposalId: string) => ReactNode
  children?: ReactNode
}

// The conversation: the person's messages on the right as typed (never
// formatted), the assistant's on the left through the safe markdown subset.
export function Thread({ messages, proposals, renderProposal, children }: Props) {
  return (
    <ol className="flex flex-col gap-3">
      {threadItems(messages).map(item => item.kind === 'tools'
        ? <ToolRun key={`t-${item.key}`} names={item.names} />
        : <MessageItem key={item.message.seq} message={item.message} proposals={proposals} renderProposal={renderProposal} />)}
      {children}
    </ol>
  )
}

function ToolRun({ names }: { names: string[] }) {
  const { t } = useTranslation('assistant')
  const [open, setOpen] = useState(false)
  return (
    <li className="self-start flex flex-col gap-1 pl-1">
      <button type="button" onClick={() => setOpen(o => !o)} aria-expanded={open} className="flex items-center gap-1 text-2xs text-ink-4 hover:text-ink-3">
        <ChevronRight size={11} className={clsx('transition-transform', open && 'rotate-90')} aria-hidden />
        {t('conversation.lookedUpCount', { count: names.length })}
      </button>
      {open && (
        <ul className="pl-4 flex flex-col gap-0.5">
          {names.map((n, i) => <li key={i} className="text-2xs text-ink-4 font-mono">{n}</li>)}
        </ul>
      )}
    </li>
  )
}

function MessageItem({ message: m, proposals, renderProposal }: {
  message: AssistantMessage
  proposals: AssistantProposal[]
  renderProposal?: (proposalId: string) => ReactNode
}) {
  const { t } = useTranslation('assistant')

  if (m.notice === 'proposed' && m.proposal_id && renderProposal) {
    const card = renderProposal(m.proposal_id)
    if (card) return card
  }
  if (m.notice === 'proposal_confirmed' || m.notice === 'proposal_rejected') {
    return (
      <li data-testid={`assistant-notice-${m.seq}`} className="self-center text-2xs text-ink-4">
        <DecisionLine notice={m.notice} proposal={proposals.find(p => p.id === m.proposal_id)} />
      </li>
    )
  }
  const text = m.notice ? t(`notice.${m.notice}`) : m.content
  if (!text) return null

  const mine = m.role === 'user'
  return (
    <li
      data-testid={`assistant-message-${m.seq}`} data-question={m.question ? 'true' : 'false'}
      className={clsx('flex flex-col gap-1 max-w-[85%]', mine ? 'self-end items-end' : 'self-start items-start')}
    >
      {!mine && m.question && <span className="text-2xs text-ink-4">{t('conversation.question')}</span>}
      <div
        className={clsx(
          'text-xs rounded-2xl px-3 py-2 break-words',
          mine && 'bg-accent-solid text-white whitespace-pre-wrap',
          !mine && !m.question && 'bg-sunken text-ink border border-line',
          !mine && m.question && 'bg-amber-50 dark:bg-amber-500/10 text-ink border border-amber-500/40',
        )}
      >
        {mine || m.notice ? text : <Markdown text={text} />}
      </div>
    </li>
  )
}

// What a person's decision did: "✓ Created quote Q-0012" linking to it, or
// the translated line when there's nothing to link.
function DecisionLine({ notice, proposal: p }: { notice: 'proposal_confirmed' | 'proposal_rejected'; proposal?: AssistantProposal }) {
  const { t } = useTranslation('assistant')
  if (notice === 'proposal_confirmed' && p?.status === 'confirmed' && p.result_id) {
    const order = p.kind === 'convert_quote'
    const label = order ? t('proposal.createdOrder', { ref: p.result_ref ?? '' }) : t('proposal.createdQuote', { ref: p.result_ref ?? '' })
    return (
      <Link to={order ? salesOrderPath(p.result_id) : quotePath(p.result_id)} className="text-success hover:underline">
        {`✓ ${label}`}
      </Link>
    )
  }
  return <>{t(`notice.${notice}`)}</>
}
