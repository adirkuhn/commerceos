import { Loader2, RotateCw } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../../format/DateFormatProvider'
import type { AssistantRequest } from '../types'

interface Props {
  request: AssistantRequest
  retrying: boolean
  onRetry: () => void
}

// The assistant's side of the thread while it isn't answering yet:
// thinking, waiting its turn, or waiting after a failed attempt -- then with
// why, when it tries next, the error, and Retry now.
export function WaitingStatus({ request, retrying, onRetry }: Props) {
  const { t } = useTranslation('assistant')
  const { formatTime } = useDateFormat()
  const [details, setDetails] = useState(false)

  if (request.status === 'processing') {
    return <Bubble><Loader2 size={12} className="animate-spin" aria-hidden />{t('conversation.thinking')}</Bubble>
  }
  if (request.status !== 'queued') return null
  if (!request.wait_reason) {
    return <Bubble><Loader2 size={12} className="animate-spin" aria-hidden />{t('conversation.queued')}</Bubble>
  }
  return (
    <li className="self-start max-w-[85%] flex flex-col gap-1.5 rounded-xl border border-amber-500/40 bg-amber-50 dark:bg-amber-500/10 px-3 py-2">
      <p className="text-xs text-warning">{t(`wait.${request.wait_reason}`)}</p>
      <p className="text-2xs text-ink-4">{t('wait.nextTry', { time: formatTime(request.next_attempt_at) })}</p>
      {request.last_error && (
        <>
          <button type="button" onClick={() => setDetails(d => !d)} aria-expanded={details} className="self-start text-2xs text-ink-3 hover:text-ink underline">
            {details ? t('wait.hideDetails') : t('wait.showDetails')}
          </button>
          {details && <p className="text-2xs text-ink-4 font-mono break-words whitespace-pre-wrap">{request.last_error}</p>}
        </>
      )}
      <button
        type="button" onClick={onRetry} disabled={retrying}
        className="self-start flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-line bg-surface text-xs text-ink-2 hover:text-ink disabled:opacity-50"
      >
        <RotateCw size={12} className={retrying ? 'animate-spin' : undefined} aria-hidden />
        {retrying ? t('chat.retrying') : t('chat.retryNow')}
      </button>
    </li>
  )
}

function Bubble({ children }: { children: ReactNode }) {
  return (
    <li className="self-start flex items-center gap-1.5 rounded-xl border border-line bg-sunken px-3 py-2 text-xs text-ink-3">
      {children}
    </li>
  )
}
