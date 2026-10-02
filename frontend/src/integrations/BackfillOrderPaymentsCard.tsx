import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { useBackfillOrderPayments } from './hooks/useBackfillOrderPayments'

// "Fetch missing payment info" for a WooCommerce connection: fills in the
// transaction id / paid date on orders imported before they were paid.
export function BackfillOrderPaymentsCard({ token, connectionId }: { token: string; connectionId: string }) {
  const { t } = useTranslation('integrations')
  const { run, progress, running, done, error } = useBackfillOrderPayments(token, connectionId)

  return (
    <div className="bg-raised rounded-xl border border-line p-4">
      <h2 className="text-sm font-semibold text-ink mb-1">{t('woo.backfillPaymentsTitle')}</h2>
      <p className="text-xs text-ink-4 mb-3">{t('woo.backfillPaymentsIntro')}</p>
      <button
        type="button" onClick={() => { void run() }} disabled={running}
        className="px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
      >
        {t('woo.backfillPayments')}
      </button>
      {running && (
        <p className="mt-2 text-2xs text-ink-3 flex items-center gap-1.5">
          <Loader2 size={12} className="animate-spin" /> {t('woo.backfillPaymentsRunning', { checked: progress.checked, updated: progress.updated })}
        </p>
      )}
      {done && (
        <p className="mt-2 text-2xs text-success">
          {progress.checked === 0
            ? t('woo.backfillPaymentsNothing')
            : t('woo.backfillPaymentsDone', { checked: progress.checked, updated: progress.updated })}
        </p>
      )}
      {error && <p className="mt-2 text-2xs text-danger">{error}</p>}
      {progress.errors.length > 0 && !running && (
        <div className="mt-2 text-2xs space-y-0.5">
          <p className="text-ink-3">{t('woo.backfillPaymentsErrors')}</p>
          {progress.errors.map((e, i) => <p key={i} className="text-danger">{e}</p>)}
        </div>
      )}
    </div>
  )
}
