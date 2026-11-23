import clsx from 'clsx'
import { MoreHorizontal, Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useDateFormat } from '../../format/DateFormatProvider'
import { assistantRequestPath, isCancellable } from '../format'
import type { AssistantRequestsState } from '../hooks/useAssistantRequests'
import type { AssistantRequest, AssistantRequestStatus } from '../types'
import { WaitReason } from '../WaitReason'
import { relativeTime } from './relativeTime'

const DOT: Record<AssistantRequestStatus, string> = {
  queued: 'bg-ink-4',
  processing: 'bg-accent animate-pulse',
  needs_input: 'bg-amber-500',
  awaiting_confirmation: 'bg-amber-500',
  answered: 'bg-green-500',
  cancelled: 'bg-line',
}

// A queued request that already failed an attempt can be tried again now.
export function isRetryable(r: AssistantRequest): boolean {
  return r.status === 'queued' && !!r.wait_reason
}

interface Props {
  list: AssistantRequestsState
  activeId: string | null
  onNewChat: () => void
  onOpen: () => void
  onCancel: (id: string) => void
  onRetry: (id: string) => void
  className?: string
}

// The chats, newest first: open one, start a new one, or cancel / retry
// from a row's menu.
export function ChatHistory({ list, activeId, onNewChat, onOpen, onCancel, onRetry, className }: Props) {
  const { t } = useTranslation(['assistant', 'common'])
  return (
    <nav aria-label={t('chat.history')} className={clsx('flex-col w-72 shrink-0 border-r border-line bg-sunken min-h-0', className)}>
      <div className="p-3 shrink-0">
        <button
          type="button" onClick={onNewChat}
          className="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-surface text-xs text-ink-2 hover:text-ink"
        >
          <Plus size={13} aria-hidden /> {t('chat.newChat')}
        </button>
      </div>
      <div className="flex-1 min-h-0 overflow-auto px-2 pb-3 flex flex-col gap-1">
        {list.error && <p className="px-2 text-xs text-danger">{list.error}</p>}
        {!list.loading && !list.error && list.items.length === 0 && <p className="px-2 text-xs text-ink-4">{t('chat.empty')}</p>}
        <ul className="flex flex-col gap-0.5">
          {list.items.map(r => (
            <HistoryRow key={r.id} request={r} active={r.id === activeId} onOpen={onOpen} onCancel={() => onCancel(r.id)} onRetry={() => onRetry(r.id)} />
          ))}
        </ul>
        {list.loading && <p className="px-2 text-xs text-ink-4">{t('common:status.loading')}</p>}
        {list.hasMore && !list.loading && (
          <button type="button" onClick={list.loadMore} className="self-start mx-2 px-2.5 py-1 rounded-lg bg-fill text-ink-3 text-xs hover:text-ink-2">
            {t('list.loadMore')}
          </button>
        )}
      </div>
    </nav>
  )
}

function HistoryRow({ request: r, active, onOpen, onCancel, onRetry }: {
  request: AssistantRequest
  active: boolean
  onOpen: () => void
  onCancel: () => void
  onRetry: () => void
}) {
  const { t, i18n } = useTranslation('assistant')
  const { formatDate } = useDateFormat()
  const [menu, setMenu] = useState(false)
  const retryable = isRetryable(r)
  const cancellable = isCancellable(r.status)

  return (
    <li className={clsx('relative group flex items-start gap-1 rounded-lg', active ? 'bg-hover' : 'hover:bg-hover')}>
      <Link
        to={assistantRequestPath(r.id)} onClick={onOpen} aria-current={active ? 'page' : undefined}
        className="flex-1 min-w-0 flex flex-col gap-0.5 px-2 py-1.5"
      >
        <span className="flex items-center gap-1.5 min-w-0">
          <span className={clsx('w-1.5 h-1.5 rounded-full shrink-0', DOT[r.status])} aria-hidden />
          <span className="text-xs text-ink truncate">{r.title}</span>
        </span>
        <span className="text-2xs text-ink-4 pl-3">
          {relativeTime(r.created_at, i18n.language, new Date(), formatDate)}
          {" "}<span className="sr-only">{t(`status.${r.status}`)}</span>
        </span>
        <span className="pl-3"><WaitReason request={r} /></span>
      </Link>
      {(retryable || cancellable) && (
        <button
          type="button" onClick={() => setMenu(m => !m)} aria-haspopup="menu" aria-expanded={menu}
          aria-label={t('chat.actionsFor', { title: r.title })}
          className="shrink-0 mt-1 mr-1 p-1 rounded text-ink-4 hover:text-ink opacity-70 group-hover:opacity-100"
        >
          <MoreHorizontal size={13} aria-hidden />
        </button>
      )}
      {menu && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setMenu(false)} />
          <div role="menu" className="absolute right-1 top-7 z-50 min-w-32 rounded-lg border border-line bg-raised shadow-xl p-1"
            onKeyDown={e => { if (e.key === 'Escape') setMenu(false) }}>
            {retryable && (
              <button type="button" role="menuitem" onClick={() => { setMenu(false); onRetry() }}
                className="w-full text-left px-2.5 py-1.5 rounded text-xs text-ink-2 hover:bg-hover">
                {t('chat.retryNow')}
              </button>
            )}
            {cancellable && (
              <button type="button" role="menuitem" onClick={() => { setMenu(false); onCancel() }}
                className="w-full text-left px-2.5 py-1.5 rounded text-xs text-danger hover:bg-hover">
                {t('list.cancel')}
              </button>
            )}
          </div>
        </>
      )}
    </li>
  )
}
