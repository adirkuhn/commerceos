import { useState, type ReactNode } from 'react'
import { ArrowLeft, AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { apiErrorText } from '../api/apiErrorText'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { useResource } from '../routing/useResource'
import { useStackNavigation } from '../routing/useStackNavigation'
import * as api from './api'
import { DEAD_LETTERS_PATH, prettyData } from './format'
import type { DeadLetter, DeadLetterStream } from './types'

type Busy = 'retry' | 'discard' | null

// DeadLetterDetail is one dead letter: its failure details, the event data
// pretty-printed, and Retry / Discard -- both confirmed first. After either
// succeeds it returns to the list it was opened from.
export function DeadLetterDetail({ stream, seq }: { stream: DeadLetterStream; seq: number }) {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = useAuth()
  const { exitToOpener } = useStackNavigation()
  const { confirm, confirmDialog } = useConfirm()
  const [busy, setBusy] = useState<Busy>(null)
  const [actionError, setActionError] = useState('')
  const { data, error, loading } = useResource(() => api.getDeadLetter(token ?? '', stream, seq), [token, stream, seq])

  const back = () => exitToOpener(`${DEAD_LETTERS_PATH}?stream=${stream}`)

  async function run(kind: Exclude<Busy, null>, d: DeadLetter) {
    const warning = t('deadLetters.retryWarning')
    const ok = await confirm(kind === 'retry'
      ? {
          title: t('deadLetters.confirmRetry.title'),
          message: d.consumer_known
            ? t('deadLetters.confirmRetry.messageFor', { consumer: d.consumer, warning })
            : t('deadLetters.confirmRetry.messageAll', { subject: d.subject, warning }),
          confirmLabel: t('deadLetters.confirmRetry.confirm'),
        }
      : {
          title: t('deadLetters.confirmDiscard.title'),
          message: t('deadLetters.confirmDiscard.message'),
          confirmLabel: t('deadLetters.confirmDiscard.confirm'),
          destructive: true,
        })
    if (!ok) return
    setBusy(kind)
    setActionError('')
    try {
      if (kind === 'retry') await api.retryDeadLetter(token ?? '', stream, seq)
      else await api.discardDeadLetter(token ?? '', stream, seq)
      back()
    } catch (err) {
      setActionError(apiErrorText(err) ?? t(kind === 'retry' ? 'deadLetters.retryFailed' : 'deadLetters.discardFailed'))
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="p-5 flex flex-col gap-4 max-w-4xl">
      <button type="button" onClick={back} className="self-start flex items-center gap-1 text-xs text-ink-3 hover:text-ink-2">
        <ArrowLeft size={12} /> {t('deadLetters.back')}
      </button>

      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {error && <p className="text-xs text-danger">{error}</p>}

      {data && (
        <>
          <div className="flex items-start justify-between gap-3">
            <h2 className="text-sm font-semibold text-ink">{t('deadLetters.detailTitle', { seq: data.sequence })}</h2>
            <div className="flex gap-2">
              <button
                type="button" onClick={() => run('retry', data)} disabled={busy !== null}
                className="px-2.5 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
              >
                {busy === 'retry' ? t('deadLetters.retrying') : t('deadLetters.retry')}
              </button>
              <button
                type="button" onClick={() => run('discard', data)} disabled={busy !== null}
                className="px-2.5 py-1.5 rounded-lg bg-fill text-danger text-xs font-medium hover:bg-hover disabled:opacity-50"
              >
                {busy === 'discard' ? t('deadLetters.discarding') : t('deadLetters.discard')}
              </button>
            </div>
          </div>

          <p className="flex gap-1.5 text-xs text-ink-3 bg-sunken rounded-lg p-2.5">
            <AlertTriangle size={14} className="shrink-0 text-warning" aria-hidden />
            {t('deadLetters.retryWarning')}
          </p>

          {actionError && <p className="text-xs text-danger">{actionError}</p>}

          <Fields d={data} />

          <section>
            <h3 className="text-xs font-semibold caps-label text-ink-3 mb-2">{t('deadLetters.data')}</h3>
            <pre data-testid="dead-letter-data" className="whitespace-pre-wrap break-all text-2xs font-mono bg-sunken rounded-lg p-3 max-h-[480px] overflow-y-auto text-ink-2">
              {prettyData(data.data)}
            </pre>
          </section>
        </>
      )}
      {confirmDialog}
    </div>
  )
}

function Fields({ d }: { d: DeadLetter }) {
  const { t } = useTranslation('settings')
  const { formatDateTime } = useDateFormat()
  const rows: [string, ReactNode][] = [
    [t('deadLetters.fields.stream'), t(`deadLetters.streams.${d.stream}`)],
    [t('deadLetters.fields.subject'), <span className="font-mono">{d.subject}</span>],
    [t('deadLetters.fields.consumer'), d.consumer_known
      ? d.consumer
      : <span className="italic text-ink-4">{t('deadLetters.unknownConsumer')} — {t('deadLetters.unknownConsumerHint')}</span>],
    [t('deadLetters.fields.eventId'), <span className="font-mono">{d.event_id || '—'}</span>],
    [t('deadLetters.fields.tenant'), <span className="font-mono">{d.tenant_id || '—'}</span>],
    [t('deadLetters.fields.error'), <span className="text-danger">{d.error || '—'}</span>],
    [t('deadLetters.fields.deliveries'), d.num_delivered || '—'],
    [t('deadLetters.fields.deadLetteredAt'), d.dead_lettered_at ? formatDateTime(d.dead_lettered_at) : '—'],
    [t('deadLetters.fields.storedAt'), formatDateTime(d.stored_at)],
  ]
  return (
    <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5 text-xs">
      {rows.map(([label, value]) => (
        <div key={label} className="contents">
          <dt className="text-ink-4">{label}</dt>
          <dd className="text-ink-2 break-all">{value}</dd>
        </div>
      ))}
    </dl>
  )
}
