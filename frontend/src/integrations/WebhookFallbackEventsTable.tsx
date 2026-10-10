import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import Pagination from '../components/common/Pagination'
import { useWebhookFallbackEvents } from './hooks/useWebhookFallbackEvents'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

export function WebhookFallbackEventsTable({
  events, total, page, perPage, loading, error, setPage, retry, abandon, abandonAll,
}: ReturnType<typeof useWebhookFallbackEvents>) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['integrations', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const [rowError, setRowError] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [abandoningAll, setAbandoningAll] = useState(false)

  async function handleRetry(id: string) {
    setRowError('')
    setBusyId(id)
    try {
      await retry(id)
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('fallback.retryFailed'))
    } finally {
      setBusyId(null)
    }
  }

  async function handleAbandon(id: string) {
    setRowError('')
    setBusyId(id)
    try {
      await abandon(id)
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('fallback.abandonFailed'))
    } finally {
      setBusyId(null)
    }
  }

  async function handleAbandonAll() {
    if (!(await confirm({ title: t('fallback.abandonAllTitle', { count: total }), message: t('fallback.abandonAllMessage'), confirmLabel: t('fallback.abandonAllConfirm'), destructive: true }))) {
      return
    }
    setRowError('')
    setAbandoningAll(true)
    try {
      await abandonAll()
    } catch (err) {
      setRowError(apiErrorText(err) ?? t('fallback.abandonAllFailed'))
    } finally {
      setAbandoningAll(false)
    }
  }

  if (!loading && events.length === 0 && total === 0 && !error) return null

  return (
    <div className="bg-raised rounded-xl border border-line p-4">
      <div className="flex items-start justify-between gap-3 mb-1">
        <h2 className="text-sm font-semibold text-ink">{t('fallback.title')}</h2>
        {total > 0 && (
          <button
            onClick={handleAbandonAll}
            disabled={abandoningAll}
            className="shrink-0 px-2 py-1 rounded text-2xs font-medium bg-fill text-ink-3 hover:text-danger disabled:opacity-50 transition-colors"
          >
            {abandoningAll ? t('fallback.abandoning') : t('fallback.abandonAll')}
          </button>
        )}
      </div>
      <p className="text-xs text-ink-4 mb-3">
        {t('fallback.intro')}
      </p>

      {error && <p className="text-xs text-danger mb-2">{error}</p>}
      {rowError && <p className="text-xs text-danger mb-2">{rowError}</p>}

      {loading && events.length === 0 ? (
        <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
      ) : events.length === 0 ? (
        <p className="text-xs text-ink-4">{t('fallback.empty')}</p>
      ) : (
        <div className="max-h-96 overflow-y-auto overflow-x-auto border border-line rounded-lg">
          <table className="w-full text-xs">
            <thead className="sticky top-0 bg-raised">
              <tr className="text-left text-ink-4 border-b border-line">
                <th className="py-2 pr-3 pl-3 font-medium">{t('fallback.provider')}</th>
                <th className="py-2 pr-3 font-medium">{t('fallback.eventId')}</th>
                <th className="py-2 pr-3 font-medium">{t('fallback.received')}</th>
                <th className="py-2 pr-3 font-medium">{t('common:fields.status')}</th>
                <th className="py-2 pr-3 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {events.map(ev => (
                <tr key={ev.id} className="border-b border-line last:border-0">
                  <td className="py-2 pr-3 pl-3 text-ink-2 capitalize">{ev.provider}</td>
                  <td className="py-2 pr-3 text-ink-3 font-mono truncate max-w-[160px]" title={ev.event_id}>{ev.event_id}</td>
                  <td className="py-2 pr-3 text-ink-3">{formatDateTime(ev.received_at)}</td>
                  <td className="py-2 pr-3 text-ink-3">{ev.status}</td>
                  <td className="py-2 pr-3">
                    <div className="flex gap-2 justify-end">
                      <button
                        onClick={() => handleRetry(ev.id)}
                        disabled={busyId === ev.id}
                        className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-accent disabled:opacity-50 transition-colors"
                      >
                        {t('fallback.retry')}
                      </button>
                      <button
                        onClick={() => handleAbandon(ev.id)}
                        disabled={busyId === ev.id}
                        className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-danger disabled:opacity-50 transition-colors"
                      >
                        {t('fallback.abandon')}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Pagination page={page} pageSize={perPage} totalItems={total} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}
