import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useDateFormat } from '../format/DateFormatProvider'
import { useQueryEnum } from '../routing/useQueryState'
import { useStackNavigation } from '../routing/useStackNavigation'
import { assistantRequestPath, isCancellable } from './format'
import { useAssistantActions } from './hooks/useAssistantActions'
import { useAssistantRequests } from './hooks/useAssistantRequests'
import { StatusChip } from './StatusChip'
import { ASSISTANT_STATUSES, type AssistantRequest, type AssistantRequestStatus } from './types'
import { useCancelRequest } from './useCancelRequest'
import { WaitReason } from './WaitReason'

type StatusFilter = AssistantRequestStatus | 'all'
const FILTERS: readonly StatusFilter[] = ['all', ...ASSISTANT_STATUSES]

// /assistant: write or paste a request (it opens its own page at once and
// waits there), and every request the tenant has asked.
export function AssistantHome() {
  const { t } = useTranslation(['assistant', 'common'])
  const { token } = useAuth()
  const { open } = useStackNavigation()
  const [text, setText] = useState('')
  const [filter, setFilter] = useQueryEnum<StatusFilter>('status', FILTERS, 'all')
  const list = useAssistantRequests(token ?? '', filter === 'all' ? undefined : filter)
  const actions = useAssistantActions(token ?? '')
  const { cancel, confirmDialog } = useCancelRequest(actions, list.replace)

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!text.trim()) return
    const created = await actions.create(text)
    if (created) open(assistantRequestPath(created.id))
  }

  return (
    <div className="p-5 flex flex-col gap-5 max-w-3xl">
      <p className="text-xs text-ink-4">{t('intro')}</p>

      <form onSubmit={submit} className="flex flex-col gap-2">
        <label htmlFor="assistant-text" className="text-2xs text-ink-4 caps-label">{t('compose.label')}</label>
        <textarea
          id="assistant-text" rows={5} value={text} onChange={e => setText(e.target.value)} maxLength={20000}
          placeholder={t('compose.placeholder')}
          className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus resize-y"
        />
        {actions.error && <p className="text-xs text-danger">{actions.error}</p>}
        <div>
          <button
            type="submit" disabled={actions.busy || !text.trim()}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {actions.busy ? t('compose.sending') : t('compose.submit')}
          </button>
        </div>
      </form>

      <section className="flex flex-col gap-2">
        <div className="flex items-end justify-between gap-2">
          <h2 className="text-sm font-semibold text-ink">{t('list.title')}</h2>
          <div>
            <label htmlFor="assistant-filter" className="block text-2xs text-ink-4 mb-1">{t('list.filter')}</label>
            <select
              id="assistant-filter" value={filter} onChange={e => setFilter(e.target.value as StatusFilter)}
              className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink-2"
            >
              {FILTERS.map(f => <option key={f} value={f}>{f === 'all' ? t('list.all') : t(`status.${f}`)}</option>)}
            </select>
          </div>
        </div>

        {list.error && <p className="text-xs text-danger">{list.error}</p>}
        {!list.loading && !list.error && list.items.length === 0 && (
          <p className="text-xs text-ink-4">{filter === 'all' ? t('list.empty') : t('list.emptyFiltered')}</p>
        )}
        <ul className="flex flex-col divide-y divide-line border border-line rounded-lg">
          {list.items.map(r => (
            <RequestRow key={r.id} request={r} onOpen={() => open(assistantRequestPath(r.id))} onCancel={() => cancel(r.id)} />
          ))}
        </ul>
        {list.loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
        {list.hasMore && !list.loading && (
          <button type="button" onClick={list.loadMore} className="self-start px-2.5 py-1.5 rounded-lg bg-fill text-ink-3 text-xs hover:text-ink-2">
            {t('list.loadMore')}
          </button>
        )}
      </section>
      {confirmDialog}
    </div>
  )
}

function RequestRow({ request, onOpen, onCancel }: { request: AssistantRequest; onOpen: () => void; onCancel: () => void }) {
  const { t } = useTranslation('assistant')
  const { formatDateTime } = useDateFormat()
  return (
    <li className="flex items-start gap-3 px-3 py-2.5 hover:bg-hover">
      <div className="min-w-0 flex-1 flex flex-col gap-1">
        <a
          href={assistantRequestPath(request.id)} onClick={e => { e.preventDefault(); onOpen() }}
          className="text-xs text-ink font-medium hover:text-accent truncate"
        >
          {request.title}
        </a>
        <span className="text-2xs text-ink-4">{t('list.asked')} {formatDateTime(request.created_at)}</span>
        <WaitReason request={request} />
      </div>
      <StatusChip status={request.status} />
      {isCancellable(request.status) && (
        <button type="button" onClick={onCancel} className="text-2xs text-ink-4 hover:text-danger">{t('list.cancel')}</button>
      )}
    </li>
  )
}
